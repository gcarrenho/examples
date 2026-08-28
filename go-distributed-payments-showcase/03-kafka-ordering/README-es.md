> 🌐 [English](README.md)

# Caso 03 — Orden por Partición en Kafka + Dead Letter Queue

## El Problema de Ordenamiento

Kafka garantiza **entrega en orden dentro de una partición**, no en todo el topic.

Para eventos de pagos, esto crea una restricción dura: un REEMBOLSO debe procesarse
*después* del COBRO original, o el reembolso tocará un estado de cuenta que todavía
no existe.

Si los eventos caen en particiones distintas:

```
Partición 0: [REEMBOLSO  ← procesado por consumer-1 primero]
Partición 1: [COBRO      ← procesado por consumer-2, llega después]
```

Consumer-1 procesa el REEMBOLSO contra una cuenta que aún no tiene ningún cobro — estado inválido.

## Solución 1: Asignación Determinista de Partición

Rutar todos los eventos de la misma cuenta a la misma partición usando un hash determinista:

```
partición = SHA-256(account_id)[0:4] mod numPartitions
```

SHA-256 provee distribución uniforme — sin particiones calientes incluso cuando los
IDs de cuenta comparten un prefijo común (ej: `acc-0000001`, `acc-0000002`, ...).

La misma cuenta siempre cae en la misma partición **para siempre**, sin importar cuál
instancia del productor envía el mensaje ni cuántas veces se reinicia el servicio.

## Solución 2: Consumidor Síncrono por Partición

Dentro de una partición, el consumidor procesa un mensaje a la vez (sin goroutines internas).
El paralelismo entre particiones se logra con **una goroutine por partición asignada**:

```
Goroutine Partición 0: [msg@0] → [msg@1] → [msg@2] → ...   (secuencial)
Goroutine Partición 1: [msg@0] → [msg@1] → [msg@2] → ...   (secuencial)
Goroutine Partición 2: [msg@0] → [msg@1] → [msg@2] → ...   (secuencial)
```

## El Patrón Dead Letter Queue (DLQ)

Algunos mensajes no pueden procesarse — JSON malformado, servicio downstream
permanentemente no disponible, tipo de evento desconocido. Reintentar para siempre
bloquea la partición: **un mensaje envenenado detiene todo el stream de eventos**.

La DLQ rutea los mensajes no procesables a un topic separado y confirma el offset
original, permitiendo que la partición avance más allá del mensaje envenenado.

```
                    ┌──────────┐
Mensaje normal  →   │ Handler  │→ ResultOK    → confirmar offset
                    │          │→ ResultRetry → no confirmar (Kafka re-entrega)
                    │          │→ ResultDLQ   → escribir al topic DLQ → confirmar offset
                    └──────────┘
```

## ¿Por Qué No Usar Kafka? Ver el Caso 05

Si tu equipo está en AWS o no quiere gestionar un cluster de Kafka, el
[Caso 05](../05-sqs-ordering/) muestra cómo SQS FIFO resuelve el mismo problema
de ordenamiento con infraestructura completamente gestionada.

## Ejecutar los Tests

```bash
go test ./03-kafka-ordering/... -race -v -count=1
```

## ¿Qué Pasa Si el Pod se Cae Durante la Operación?

**Escenario**: el consumidor procesa un mensaje y se cae **antes de confirmar el offset**.
Kafka no tiene registro de que el mensaje fue procesado.

**Lo que hace Kafka automáticamente**:
1. El heartbeat de sesión se detiene → el timeout de sesión se dispara (default: ~30 s).
2. El coordinador del consumer group dispara un **rebalanceo**.
3. La partición es reasignada a otra instancia del consumidor activa.
4. El nuevo consumidor retoma desde el último **offset confirmado** — el mensaje
   no procesado es **re-entregado**.

```
Pod A: recibe [COBRO@offset=7] → ejecuta cobro → 💥 crash (offset nunca confirmado)

           timeout de sesión → rebalanceo → Pod B toma la partición

Pod B: recibe [COBRO@offset=7] → filtro idempotencia (Caso 01) → ErrAlreadyCompleted → confirma offset ✓
```

**El riesgo: procesamiento doble.** Si la lógica de negocio se ejecutó antes del crash,
el mensaje se procesa dos veces sin protección — la cuenta se debita dos veces.

**La solución: combinar con el Caso 01 (filtro de idempotencia).** Cada handler de pagos
debe consultar el filtro de idempotencia usando la clave del evento *antes* de ejecutar
el cobro. La segunda entrega choca con `ErrAlreadyCompleted` y retorna de inmediato.

**Ajustar la velocidad de recuperación**:
- `session.timeout.ms` — valores más bajos detectan crashes más rápido pero arriesgan falsos positivos bajo pausas del GC
- `heartbeat.interval.ms` — configurar en ~1/3 de `session.timeout.ms`
- `max.poll.interval.ms` — tiempo máximo entre polls; si se supera, Kafka considera muerto al consumidor
