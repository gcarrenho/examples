> 🌐 [English](README.md)

# go-distributed-payments-showcase

Un monorepo educativo en Go sobre los problemas arquitectónicos detrás de los sistemas de pagos distribuidos: idempotencia, concurrencia, entrega de eventos, ruteo, adaptadores de protocolo y orquestación de pagos bancarios. Los casos avanzan desde patrones focalizados hasta componentes realistas de un sistema de pagos.

## Por qué los pagos son arquitectónicamente interesantes

En pagos la corrección es innegociable. Un request duplicado puede cobrar dos veces; un evento reordenado puede liquidar un reembolso antes de su pago original; una falla de red transitoria puede dejar al cliente esperando una respuesta indefinidamente. Los casos hacen concretos esos fallos y muestran dónde corresponde cada mitigación.

## Casos

| # | Caso | Concepto central | Infraestructura / protocolo principal |
|---|------|------------------|---------------------------------------|
| 01 | [idempotency-distributed](01-idempotency-distributed/) | Máquina de estados Redis `SETNX` para requests duplicados | Redis |
| 02 | [concurrency-occ](02-concurrency-occ/) | Control de concurrencia optimista vs. pesimista | PostgreSQL |
| 03 | [kafka-ordering](03-kafka-ordering/) | Orden por partición y dead-letter queues | Kafka |
| 04 | [memory-syncpool](04-memory-syncpool/) | `sync.Pool` y presión de asignaciones | Ninguna |
| 05 | [sqs-ordering](05-sqs-ordering/) | Grupos FIFO, deduplicación y redrive | AWS SQS |
| 06 | [payment-processor](06-payment-processor/) | Procesador de pagos orientado a componentes | Componentes Go y adaptadores en memoria |
| 07 | [microservices](07-microservices/) | Servicios HTTP, contratos definidos por el consumidor y reservas | HTTP |
| 08 | [acquirer-gateway](08-acquirer-gateway/) | Ruteo de adquirentes, strategies y circuit breakers | Adaptadores HTTP de adquirentes |
| 09 | [psp-card-acquirer](09-psp-card-acquirer/) | Ruteo por BIN y autorizaciones de tarjeta | ISO 8583, REST |
| 10 | [core-banking-iso20022](10-core-banking-iso20022/) | Orquestación asíncrona de pagos bancarios | Kafka, Redis, ISO 20022, SWIFT MT103 |

## Estructura del repositorio

Los Casos 01–09 viven en el módulo Go raíz. El Caso 10 usa deliberadamente tres módulos independientes, coordinados localmente por `go.work`:

```text
go-distributed-payments-showcase/
├── 01-idempotency-distributed/ ... 09-psp-card-acquirer/
├── 10-core-banking-iso20022/
│   ├── banking-core/      # dominio compartido de pagos y engine OCC
│   ├── payment-api/       # API HTTP, consumer de resultados, webhook y timeout reaper
│   ├── payment-worker/    # worker Kafka idempotente y adaptadores SEPA/SWIFT
│   ├── go.work
│   └── docker-compose.yml # Redpanda y Redis para el caso
├── go.mod
└── docker-compose.yml
```

El módulo raíz tiene dependencias externas cuando un caso necesita un cliente real o una herramienta de test; los ejemplos siguen evitando frameworks de aplicación para que la arquitectura sea visible.

## Ejecutar tests

Ejecutar los casos del módulo raíz desde la raíz del repositorio:

```bash
go test ./... -race -count=1

# Benchmarks de asignaciones del Caso 04
go test ./04-memory-syncpool/... -bench=. -benchmem -count=3
go test ./04-memory-syncpool/... -bench=Parallel -benchmem -cpu=1,2,4,8
```

Ejecutar el Caso 10 desde sus propios módulos:

```bash
cd 10-core-banking-iso20022/banking-core
go test ./... -count=1

cd ../payment-api
go build ./...

cd ../payment-worker
go build ./...
```

Para ejecutar el flujo completo del Caso 10, levantar su infraestructura local y seguir su [README](10-core-banking-iso20022/README.md):

```bash
cd 10-core-banking-iso20022
docker compose up -d
```

## Principios de diseño

1. **Contratos estrechos en el límite del consumidor**: un consumidor declara solo las capacidades que necesita; los adaptadores absorben la variación de protocolos.
2. **La entrega al menos una vez requiere idempotencia**: Kafka y SQS pueden reentregar después de un crash, por lo que una clave idempotente estable debe llegar hasta el efecto lateral.
3. **Concurrencia y entrega duplicada son problemas distintos**: la idempotencia en Redis protege un evento; OCC protege actualizaciones concurrentes distintas sobre el mismo saldo.
4. **Los eventos de dominio no son payloads HTTP**: validar y traducir en el borde de entrada, y después publicar un contrato de dominio estable.
5. **Los detalles de protocolo quedan en el borde**: ISO 8583, ISO 20022, MT103 y formatos propietarios de adquirentes viven en adaptadores dedicados, nunca en la orquestación de la aplicación.
> 🌐 [English](README.md)

# go-distributed-payments-showcase

Un monorepo educativo que demuestra los **problemas arquitectónicos más difíciles** que surgen al construir sistemas de pagos distribuidos a escala. Cada caso de estudio es autocontenido, usa únicamente la librería estándar de Go y viene acompañado de tests concurrentes que prueban científicamente que la solución funciona bajo condiciones de tráfico real.

## Por Qué los Pagos Son Arquitectónicamente Interesantes

Los pagos son el dominio donde la **corrección es innegociable**. Un bug en un feed de redes sociales muestra el post incorrecto; un bug en un sistema de pagos cobra dos veces al cliente, pierde dinero, o ambas cosas a la vez. Esto obliga a los ingenieros a razonar con cuidado sobre:

- **Semántica de exactamente-una-vez** — la misma solicitud de cobro nunca debe procesarse dos veces, aunque la red la reintente 100 veces.
- **Mutación de estado concurrente** — miles de goroutines pueden intentar actualizar el mismo saldo de cuenta en simultáneo.
- **Orden de mensajes** — un reembolso nunca debe procesarse antes que el cobro original.
- **Eficiencia de memoria** — los procesadores de pagos manejan millones de solicitudes por segundo; cada asignación innecesaria agrega presión al GC y latencia en el percentil 99.

## Casos

| # | Caso | Concepto Central | Infraestructura |
|---|------|-----------------|-----------------|
| 01 | [idempotency-distributed](01-idempotency-distributed/) | Máquina de estados Redis SETNX (STARTED → COMPLETED) | Redis |
| 02 | [concurrency-occ](02-concurrency-occ/) | Control de Concurrencia Optimista (OCC) vs Pesimista (PCC) | PostgreSQL |
| 03 | [kafka-ordering](03-kafka-ordering/) | Orden por partición + Dead Letter Queue | Kafka |
| 04 | [memory-syncpool](04-memory-syncpool/) | `sync.Pool` vs asignación en heap — 1 alloc/op | Ninguna |
| 05 | [sqs-ordering](05-sqs-ordering/) | SQS FIFO como alternativa a Kafka — orden por `MessageGroupId` + DLQ via redrive | AWS SQS |

## Ejecutar los Tests

```bash
# Levantar la infraestructura (necesaria para tests de integración)
docker compose up -d

# Ejecutar todos los tests unitarios (no requieren infraestructura)
go test ./... -race -count=1 -v

# Benchmarks para el Caso 04
go test ./04-memory-syncpool/... -bench=. -benchmem -count=3

# Benchmark paralelo con distintos valores de GOMAXPROCS
go test ./04-memory-syncpool/... -bench=Parallel -benchmem -cpu=1,2,4,8
```

## Estructura del Proyecto

```
go-distributed-payments-showcase/
├── go.mod                         # Módulo único; sin dependencias externas
├── docker-compose.yml             # Postgres + Redis + Kafka para tests de integración
├── 01-idempotency-distributed/    # Máquina de estados SETNX de dos fases
│   ├── redis_idempotency.go       # Filter con interfaz StateStore
│   └── redis_idempotency_test.go  # Prueba de exactamente-una-vez con 1000 goroutines
├── 02-concurrency-occ/            # Guarda de versión OCC vs bloqueo de fila PCC
│   ├── db_occ.go                  # Ledger con estrategias OCC y PCC
│   └── db_occ_test.go             # Convergencia con 100 goroutines + contador de conflictos
├── 03-kafka-ordering/             # Ruteo determinista de particiones + DLQ
│   ├── producer.go                # Asignación de partición con SHA-256
│   ├── consumer.go                # Procesamiento síncrono + ruteo a DLQ
│   └── ordering_test.go           # Estabilidad de partición + máquina de estados del consumidor
├── 04-memory-syncpool/            # Eliminación de presión en el GC
│   ├── parser.go                  # Constructor de string canónico con []byte reciclado
│   └── parser_bench_test.go       # 1 alloc/op (Pool) vs 2 allocs/op (Direct)
└── 05-sqs-ordering/               # Alternativa a Kafka: SQS FIFO
    ├── sqs_producer.go            # MessageGroupId=AccountID + deduplicación por EventID
    ├── sqs_consumer.go            # Bucle de polling + escritura explícita a DLQ
    └── sqs_test.go                # Contratos del productor + máquina de estados del consumidor
```

## Comportamiento ante Caída de Pod: Resumen

Un pod puede caerse en cualquier momento. La tabla muestra qué hace cada componente
de infraestructura automáticamente y qué debe agregar el código de aplicación.

| Caso | Recuperación automática de la infraestructura | Responsabilidad de la aplicación |
|------|-----------------------------------------------|----------------------------------|
| 01 — Idempotencia | TTL expira → clave STARTED abandonada puede recuperarse | Usar la misma clave idempotente también en el procesador externo |
| 02 — OCC | La DB hace rollback de la escritura al cerrar la conexión (cero estado parcial) | No se necesita nada extra |
| 02 — PCC | Lock de `SELECT FOR UPDATE` liberado al cerrar la conexión (segundos) | Ajustar `tcp_keepalives_idle` para detectar conexiones muertas rápido |
| 03 — Kafka | Timeout de sesión → rebalanceo → offset no confirmado re-entregado | El handler **debe** consultar el filtro de idempotencia (Caso 01) antes de ejecutar |
| 04 — sync.Pool | N/A — el pool es memoria de scratch; el crash lo descarta sin consecuencias | El cliente reintenta el request HTTP |
| 05 — SQS FIFO | VisibilityTimeout expira → mensaje reaparece automáticamente | El handler **debe** consultar el filtro de idempotencia (Caso 01) antes de ejecutar |

> **El hilo conductor**: los Casos 03 y 05 garantizan entrega *al menos una vez*, no
> *exactamente una vez*. La garantía requiere componerlos con el Caso 01 y propagar
> la misma clave idempotente hasta el proveedor de pagos.

## Principios de Diseño

Cada caso sigue la misma estructura:

1. **Interface-first**: las dependencias externas (Redis, Postgres, Kafka) están abstraídas detrás de interfaces estrechas. Los tests unitarios corren contra implementaciones en memoria; los tests de integración corren contra infraestructura real.
2. **Sin frameworks**: solo librería estándar. Los patrones son universales — agregar una librería externa oscurecería la técnica que se quiere enseñar.
3. **Correctitud medible**: los tests usan contadores `atomic.Int64` y barreras `sync.WaitGroup` para afirmar el comportamiento científicamente, no de forma probabilística.
