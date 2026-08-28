> 🌐 [English](README.md)

# Caso 05 — SQS FIFO como Alternativa a Kafka para Eventos de Pago Ordenados

## Cuándo No Podés (o No Querés) Usar Kafka

Kafka es excelente, pero tiene un peso operacional real:

- Requiere Zookeeper o KRaft para la coordinación del cluster
- Necesita un equipo de operaciones dedicado (o una oferta gestionada como Confluent / MSK)
- Es excesivo para equipos que procesan menos de 50K mensajes por segundo
- Agrega costo de infraestructura que puede no estar justificado en las etapas tempranas de un producto

**AWS SQS FIFO** resuelve el problema de orden y exactamente-una-vez con infraestructura
completamente gestionada, cero gestión de cluster y una API HTTP simple.

## Cómo SQS FIFO Mapea a los Conceptos de Kafka

| Concepto de Kafka | Equivalente en SQS FIFO | Notas |
|-------------------|------------------------|-------|
| Topic | URL de la Cola | Una cola por stream lógico |
| Partition key | `MessageGroupId` | Mismo account_id → mismo grupo → ordenado |
| Commit de offset | `DeleteMessage` | Borrar = confirmar el éxito |
| Retención / replay | ❌ (máx 14 días, sin seek) | SQS no soporta replay de offset |
| Consumer group | Invisible via `VisibilityTimeout` | Solo un consumidor procesa un grupo a la vez |
| Topic DLQ (escritura manual) | Redrive policy (infraestructura) | Automático después de `MaxReceiveCount` reintentos |
| Clave de idempotencia | `MessageDeduplicationId` | Ventana de deduplicación integrada de 5 minutos |

## Garantía de Orden

SQS FIFO entrega mensajes dentro de un `MessageGroupId` en orden estricto y permite
solo un mensaje en vuelo por grupo a la vez:

```
Grupo "acc-1234": [COBRO@0] → [REEMBOLSO@1] → [REVERSIÓN@2]
                       ↑ procesado y borrado antes de que @1 se vuelva visible
```

Esto es estructuralmente idéntico a la garantía por partición de Kafka.

## Manejo de Fallos

```
Mensaje recibido (VisibilityTimeout = 30s)
         │
         ▼
   ┌──────────┐
   │ Handler  │ → ResultOK    → DeleteMessage (elimina de la cola)
   │          │ → ResultRetry → no hacer nada; expira el timeout; el mensaje reaparece
   │          │ → ResultDLQ   → DeleteMessage + escribir a DLQ con metadatos del error
   └──────────┘
                                     ↑
          O: dejar que expire MaxReceiveCount veces → redrive policy mueve a DLQ automáticamente
```

**Diferencia clave vs Kafka DLQ**: con Kafka escribís al topic DLQ en tu código.
Con SQS, el redrive policy se configura a nivel de cola en AWS — tu código simplemente
puede no borrar el mensaje y dejar que expire más allá de `MaxReceiveCount`.
Las escrituras explícitas a DLQ (como se muestra en este caso) agregan metadatos del
error para observabilidad.

## Kafka vs SQS FIFO — Cuándo Elegir Cada Uno

| Criterio | Kafka | SQS FIFO |
|----------|-------|----------|
| Rendimiento | Millones/seg | 3.000 msg/s por grupo; 300/s de base |
| Replay / rebobinar | ✅ Cualquier offset | ❌ Una vez borrado, desaparece |
| Infraestructura | Self-hosted o gestionado (MSK, Confluent) | Completamente gestionado (AWS) |
| Orden | Por partición | Por `MessageGroupId` |
| Productor exactamente-una-vez | ✅ (productor idempotente) | ✅ (`MessageDeduplicationId`) |
| Retención | Ilimitada (disco) | Máx 14 días |
| Modelo de costo | Costo fijo del cluster | Por solicitud (USD 0,50 / millón) |
| Mejor para | Streams de alto volumen, analytics, replay | Colas de tareas de volumen moderado, stacks nativos de AWS |

## Cuándo Elegir SQS FIFO sobre Kafka

Elegí SQS FIFO cuando:
- Ya estás en AWS y querés reducir la complejidad operacional
- Tu volumen de mensajes es moderado (< 300K msg/s en total)
- No necesitás replay de mensajes históricos
- Querés que AWS gestione durabilidad, particionado y escalado
- El costo del cluster de Kafka supera el costo por solicitud de SQS a tu escala

Seguí con Kafka cuando:
- Necesitás replay y reprocessing de datos históricos
- Tu throughput supera los límites de FIFO por grupo
- Ya tenés experiencia operacional con Kafka en tu equipo
- Usás Kafka Streams, ksqlDB u otras herramientas del ecosistema

## Ejecutar los Tests

```bash
go test ./05-sqs-ordering/... -race -v -count=1
```

## ¿Qué Pasa Si el Pod se Cae Durante la Operación?

**Escenario**: el consumidor llama a `Receive()` (SQS arranca el reloj del
`VisibilityTimeout`), empieza a procesar y se cae antes de llamar a `DeleteMessage()`.

**Lo que hace SQS automáticamente** — no se necesita código:
1. El `VisibilityTimeout` expira (configurable, default 30 s).
2. El mensaje **vuelve a ser visible en la cola**.
3. Otro consumidor (o el pod recuperado) lo recibe y procesa.

```
Pod A: Receive [COBRO] → ejecuta cobro → 💥 crash (DeleteMessage nunca llamado)

           VisibilityTimeout expira (ej: 30 s) → mensaje reaparece

Pod B: Receive [COBRO] → filtro idempotencia (Caso 01) → ErrAlreadyCompleted → DeleteMessage ✓
```

Esta es la ventaja estructural de SQS sobre Kafka para la recuperación de crashes:
no hay timeout de sesión, ni rebalanceo, ni reasignación de particiones. El modelo
de visibilidad maneja la re-entrega automáticamente a nivel del broker.

**El riesgo: procesamiento doble** (igual que el Caso 03). Aplicar el filtro de
idempotencia del Caso 01 en cada handler de SQS antes de ejecutar el cobro.

**Ajustar el `VisibilityTimeout`**: configurarlo un poco más largo que el tiempo
máximo de procesamiento. Demasiado corto → re-entregas falsas bajo carga normal;
demasiado largo → recuperación lenta después de un crash.
