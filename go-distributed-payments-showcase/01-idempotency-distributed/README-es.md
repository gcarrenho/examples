> 🌐 [English](README.md)

# Caso 01 — Filtro de Idempotencia Distribuido

## El Problema

Una app de banca móvil reintenta una solicitud de pago fallida 3 veces por un timeout de red.
El procesador de pagos recibe 3 solicitudes idénticas. Sin un filtro, el cliente es cobrado 3 veces.

## La Solución: Máquina de Estados con SETNX

```
El cliente envía la solicitud con idempotency_key: "txn-abc123"

Solicitud 1 → SETNX("txn-abc123", "STARTED") → clave insertada → procesar pago → SET("txn-abc123", "COMPLETED")
Solicitud 2 → SETNX(...) pierde → estado STARTED → retornar ErrDuplicate o esperar y consultar
Solicitud 3 → igual que la Solicitud 2; si ya terminó, recibe ErrAlreadyCompleted
```

`SETNX` ("SET if Not eXists") es **atómica** — lee y escribe en una sola operación
sin ninguna ventana de tiempo donde una condición de carrera sea posible.

## Máquina de Estados

```
             ┌─────────────┐
             │  (sin clave)│ ◄── Release() ante fallo transitorio (habilita reintento)
             └──────┬──────┘
                    │ Acquire() — SETNX gana la carrera
                    ▼
             ┌─────────────┐
             │   STARTED   │ ◄── Acquire() concurrente → ErrDuplicate
             └──────┬──────┘
                    │ Complete()
                    ▼
             ┌─────────────┐
             │  COMPLETED  │ ◄── Acquire() futuro → ErrAlreadyCompleted
             └─────────────┘     (expira después del TTL)
```

## Primitivas Clave de Go

| Primitiva | Rol |
|-----------|-----|
| `sync.Mutex` | Simula la atomicidad de Redis SETNX en el store en memoria de los tests |
| `sync/atomic.Int64` | Contador de ganadores sin bloqueo en el test con 1000 goroutines |
| `chan struct{}` (cerrado) | Liberación sincronizada de goroutines para contención máxima |

## Ejecutar los Tests

```bash
go test ./01-idempotency-distributed/... -race -v -count=1
```

Con `-race`, el detector de condiciones de carrera de Go valida que el filtro en sí
no tiene races incluso cuando 1000 goroutines compiten simultáneamente.

## ¿Qué Pasa Si el Pod se Cae Durante la Operación?

Este es el escenario de fallo más crítico para el filtro de idempotencia.

**Escenario**: El Pod A llama a `Acquire()` — la clave queda en STARTED — y luego
se cae antes de llamar a `Complete()` o `Release()`.

```
Pod A: Acquire() → STARTED → 💥 crash
                                    │
                                    │ Expira el TTL
                                    ▼
Pod B: Acquire() → STARTED (fresco) → Complete() → COMPLETED ✓
```

**Sin TTL**: la clave queda en STARTED para siempre. Ningún otro pod puede procesar
la solicitud. El cliente queda permanentemente bloqueado para reintentar su pago.

**Con TTL** (ya configurado en `NewFilter`): la clave expira después de la duración
configurada y otro pod puede intentar recuperar la operación. Ver `TestFilter_PodCrashRecoveryViaTTL`.

Pero el TTL **no prueba que el pago no haya sido ejecutado**. Si Pod A cobró al
procesador externo y se cayó antes de guardar `COMPLETED`, Pod B podría repetir el
cobro cuando expire el TTL. Por eso la operación downstream también debe recibir
`txn-abc123` como clave idempotente, o Pod B debe consultar el estado del procesador
antes de volver a ejecutar:

```
Solicitud 2/3 mientras STARTED → ErrDuplicate; esperar con backoff o consultar estado
       │
       ├── estado COMPLETED → devolver la respuesta guardada, sin cobrar nuevamente
       └── STARTED + lease expirado → consultar proveedor; solo entonces recuperar
```

La estrategia correcta es **idempotencia en cada frontera**: Redis evita que dos
pods trabajen simultáneamente y el procesador de pagos evita que un retry cobre dos veces.

**Dimensionar el TTL correctamente**:

| TTL demasiado corto | TTL demasiado largo |
|---------------------|---------------------|
| Dos pods procesan el mismo pago en paralelo → cobro doble | El cliente espera mucho tiempo antes de que el reintento sea aceptado → mala UX |

Regla general: `TTL = 2–3× la duración máxima esperada del procesamiento`.
Para un pago que tarda ~2 s, usar un TTL de 5–10 s.

Los clientes y las redes mienten. Los timeouts HTTP hacen que los clientes reintentan
sin saber si el servidor recibió la solicitud original. Sin idempotencia:

- Un botón de "Pagar" presionado dos veces genera dos cargos.
- Un retry automático después de un timeout de red genera un cargo duplicado.
- Una función Lambda que se reejecutó por un error de provisión aplica la misma
  transferencia dos veces.

El filtro de idempotencia es la **primera línea de defensa** contra todos estos casos.
Los Casos 02–05 asumen que ya existe este filtro.
