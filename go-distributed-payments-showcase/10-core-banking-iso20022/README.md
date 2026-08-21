# Case 10 — Core Banking: ISO 20022 + Kafka + Multi-Módulo Go

## El sistema

Un sistema de core banking que gestiona el ciclo de vida de órdenes de pago hacia rails interbancarios. A diferencia de un PSP (Case 09) que cobra tarjetas, un core banking system mueve dinero entre cuentas bancarias reales a través de cámaras de compensación.

```
Cliente HTTP
     │
     ▼
┌───────────────────┐     Kafka topic:                 ┌──────────────────┐
│   payment-api      │──── banking.payment.orders ────▶│  payment-worker   │
│  (HTTP + Kafka     │      .initiated                  │ (Kafka consumer  │
│   Producer)        │                                  │  + OCC + rails)  │
└───────────────────┘                                   └────────┬─────────┘
     202 Accepted                                                 │
     (respuesta inmediata,                          ┌──────────────┼──────────────┐
      no espera al banco)                           ▼                             ▼
                                                  SEPA rail                    SWIFT rail
                                              (pacs.008/002, sync)          (MT103, async)                                                     │                             │
                                                     └───────────────────┐────────────────────┘
                                                                          ▼
                                          Kafka topic: banking.payment.orders.results
                                                                          │
                                                                          ▼
                                                    payment-api (consume en background,
                                                     cachea en ResultsStore)
                                                                          │
                                                                          ▼
                                            Cliente: GET /payments/{uetr} → estado final```

## Decisiones de diseño

### 1. Kafka: el productor NO llama directo al Engine

**Por qué**: si `payment-api` llamara sincrónicamente a `Engine.Initiate`, el cliente HTTP esperaría a que SEPA/SWIFT respondan — potencialmente segundos. Con Kafka de por medio, `payment-api` valida, publica el evento y devuelve `202 Accepted` en milisegundos. El worker procesa en su propio tiempo, con sus propios reintentos, sin bloquear al cliente.

```
payment-api:    valida request → construye PaymentOrderInitiated → Publish() → 202 Accepted
                                                    │
                                              Kafka topic
                                                    │
payment-worker: consume evento → idempotencia (Redis) → Engine.Initiate → SEPA/SWIFT
```

**Qué se publica en Kafka — la decisión más importante**: el evento de **dominio ya traducido** (`PaymentOrderInitiated`), nunca el JSON crudo del cliente HTTP.

| Publicar el JSON crudo del request | Publicar el evento de dominio |
|---|---|
| ❌ El worker debe re-validar y re-mapear | ✅ El worker solo orquesta el rail |
| ❌ Cambios en el contrato HTTP rompen al worker | ✅ Kafka es un contrato estable, desacoplado del HTTP |
| ❌ Cada consumidor repite la misma validación | ✅ La validación ocurre una sola vez, en el productor |

`payment-api` es dueño de la traducción `HTTP request → PaymentOrderInitiated`. Kafka transporta el evento de dominio. `payment-worker` traduce `PaymentOrderInitiated → pacs.008/MT103`. Ningún componente salta ese paso.

### 1.1. El cliente recibe `202 Accepted` — ¿cómo se entera del resultado final?

Un `202 Accepted` solo confirma que Kafka recibió el evento, **no** que el pago se completó o que había fondos suficientes. Sin un mecanismo de retorno, el cliente queda a ciegas para siempre.

La solución: un **segundo topic de resultados** (`banking.payment.orders.results`). `payment-worker` publica el estado final (`ACSC`/`RJCT`) ahí; `payment-api` lo consume en background y lo cachea en un `ResultsStore` en memoria, exponiéndolo vía `GET /payments/{uetr}`.

```
payment-worker: Engine.Initiate() → ACSC o RJCT → PublishResult() → topic .results
                                                                          │
payment-api:    consumer en background (goroutine) → ResultsStore.HandleResult()
                                                                          │
Cliente:        GET /payments/{uetr} → lee ResultsStore → 200 {status: ACSC|RJCT}
                (o 202 {status: PDNG} si aún no hay resultado)
```

**Limitación honesta del `ResultsStore` en memoria**: con una sola réplica de `payment-api` funciona bien. Con múltiples réplicas detrás de un load balancer, cada una tiene su propia copia — el cliente podría consultar una réplica que nunca vio el resultado. En producción esto se resuelve con Redis o Postgres compartido entre réplicas.

### 1.2. El bug que corregimos: liberar la clave de idempotencia en un error de negocio

La primera versión de `worker.go` trataba **todos** los errores de `Engine.Initiate` igual — liberaba la clave de idempotencia para permitir reintento:

```go
// ❌ versión original — trata fondos insuficientes como si fuera un error transitorio
if _, err := w.engine.Initiate(ctx, event.Order); err != nil {
    _ = w.idempotency.Release(ctx, key) // libera SIEMPRE
    return fmt.Errorf("worker: engine: %w", err)
}
```

`ErrInsufficientBalance` y `ErrRejectedByBank` son errores de **negocio** — nunca se resuelven solos, sin importar cuántas veces Kafka reentregue el mensaje. Liberar la clave le dice a un futuro redelivery "reintentá esto", generando un loop de fallos idénticos ante cada rebalanceo o restart del consumer.

**Corregido**: los errores de negocio marcan la clave como `COMPLETED` (terminal, no se reintenta) y publican el resultado `RJCT`. Solo los errores de infraestructura (rail caído, Redis caído) liberan la clave para reintento:

```go
result, err := w.engine.Initiate(ctx, event.Order)
switch {
case errors.Is(err, banking.ErrInsufficientBalance), errors.Is(err, banking.ErrRejectedByBank):
    _ = w.idempotency.Complete(ctx, key) // terminal — nunca reintentar
    w.publishResult(ctx, event.Order.UETR, banking.StatusRejected, err.Error())
    return nil
case err != nil:
    _ = w.idempotency.Release(ctx, key) // transitorio — permitir reintento
    return fmt.Errorf("worker: engine: %w", err)
}
```

### 2. Kafka solo no evita el doble cobro — hace falta Redis (idempotencia)

Kafka garantiza **at-least-once delivery**, nunca *exactly-once processing*. Si `payment-worker` cae justo después de llamar a SWIFT pero antes de hacer commit del offset, Kafka re-entrega el mismo mensaje. Sin protección adicional, esto es un **segundo MT103 real** al mismo beneficiario.

```
payment-worker consume evento (UETR: 550e8400-...)
       │
       ▼
Redis SETNX("payment-order:550e8400-...", "STARTED", TTL=48h)
       │
       ├── ganó el SETNX → Engine.Initiate() → SEPA/SWIFT → marca "COMPLETED"
       └── perdió el SETNX → ya STARTED o COMPLETED → skip, no reprocesa
```

Esta es la misma regla del README raíz del repo: *"Cases 03 y 05 garantizan at-least-once, no exactly-once — la garantía exactly-once requiere componerlos con Case 01 (idempotencia)"*. Case 10 con Kafka cae exactamente en esa regla.

### 3. OCC en el ledger — la segunda capa de protección concurrente

Redis evita que **el mismo evento** se procese dos veces. Pero si dos órdenes *distintas* debitan la misma cuenta al mismo tiempo (dos réplicas de `payment-worker` procesando en paralelo), hace falta Optimistic Concurrency Control — el mismo patrón del Case 02.

```go
// banking-core/engine.go
type Ledger interface {
    GetBalance(ctx, iban) (balanceCents, version int64, err error)
    Debit(ctx, iban, amountCents, expectedVersion int64) error // ErrVersionConflict si version quedó obsoleta
    Settle(ctx, orderID) error
    Reverse(ctx, orderID) error
}
```

`Engine.debitWithOCC` reintenta con `JitteredBackoff` (idéntico a `02-concurrency-occ`) cuando dos réplicas del worker chocan en la misma cuenta — sin esto, bajo alta concurrencia se produce un *thundering herd* de reintentos simultáneos.

**Las dos capas juntas dan la garantía completa:**

| Capa | Previene |
|---|---|
| Redis (idempotencia) | Procesar el mismo evento dos veces (redelivery de Kafka) |
| OCC + JitteredBackoff | Dos eventos *distintos* pisándose el balance de la misma cuenta |

### 4. Arquitectura Multi-Módulo Go — repos independientes de verdad

**El problema con un `internal/` compartido**: si `payment-api` y `payment-worker` importan el mismo `internal/banking.go`, cualquier cambio en ese archivo obliga a redeployar ambos servicios juntos. No son realmente independientes — es un monorepo disfrazado de microservicios.

**La solución**: `banking-core` es un **módulo Go separado**, versionado de forma independiente. `payment-api` y `payment-worker` lo consumen como dependencia — igual que consumirían cualquier librería de terceros.

```
10-core-banking-iso20022/
├── go.work                    ← solo para desarrollo local; NO existe en producción/CI
│
├── banking-core/               ← MÓDULO INDEPENDIENTE — go.mod propio
│   ├── go.mod                  (module github.com/examples/banking-core)
│   ├── engine.go                Engine, OCC, JitteredBackoff, Rail/Ledger (EXPORTADOS)
│   └── events.go                PaymentOrderInitiated, EventPublisher (EXPORTADOS)
│
├── payment-api/                 ← MÓDULO INDEPENDIENTE — repo propio en producción
│   ├── go.mod                   require banking-core v1.x.x
│   └── internal/
│       ├── api.go               HTTP handler → construye evento → Publish()
│       └── kafka/publisher.go
│
└── payment-worker/               ← MÓDULO INDEPENDIENTE — repo propio en producción
    ├── go.mod                    require banking-core v1.x.x
    └── internal/
        ├── worker.go             idempotencia Redis → Engine.Initiate
        ├── sepa/rail.go          implementa banking.Rail
        ├── swift/rail.go         implementa banking.Rail
        └── kafka/consumer.go
```

**Por qué `Rail`, `Ledger`, `PaymentOrderInitiated` son exportados en `banking-core`** (a diferencia de `idempotencyStore` privado en `payment-svc` del Case 07): son los puntos de extensión que **múltiples repos externos** deben poder implementar. Acá SÍ hace falta un contrato Go compartido porque los tres módulos hablan `PaymentOrder` como tipo Go, no solo JSON — a diferencia de `orders-svc`/`payment-svc` (Case 07) que se comunican exclusivamente por HTTP y nunca comparten un tipo Go.

**El truco de desarrollo local — `go.work` + `replace`:**

```go
// payment-api/go.mod
require github.com/examples/banking-core v0.0.0-00010101000000-000000000000
replace github.com/examples/banking-core => ../banking-core
```

En desarrollo, el `go.work` (o el `replace`) apunta a la carpeta local — no hace falta publicar `banking-core` a cada cambio. En producción real:
1. `banking-core` se publica a un registry Go interno (o tag de Git) con su propia versión: `v1.2.0`
2. `payment-api` y `payment-worker` viven en **repos Git separados**
3. Cada uno declara `require github.com/miempresa/banking-core v1.2.0` sin ningún `replace`
4. Actualizar `banking-core` es un `go get -u` independiente en cada servicio, en su propio momento

### 5. ISO 20022 vive en `payment-worker`, no en `banking-core`

`iso20022/` (los tipos XML de `pacs.008`/`pacs.002`) es responsabilidad exclusiva de `payment-worker` porque es **detalle de protocolo del rail SEPA**, no del dominio de pagos. `banking-core` nunca importa `encoding/xml`.

```
banking-core/engine.go   → habla PaymentOrder (dominio)
payment-worker/sepa/     → traduce PaymentOrder ↔ pacs.008/pacs.002 (protocolo)
payment-worker/swift/    → traduce PaymentOrder ↔ MT103 (protocolo)
```

Si mañana aparece SPEI (México), se agrega `payment-worker/spei/rail.go` implementando `banking.Rail` — cero cambios en `banking-core`.

### 6. SEPA vs SWIFT: protocolos completamente distintos

| | SEPA Credit Transfer | SWIFT MT103 |
|---|---|---|
| **Formato** | XML (ISO 20022 pacs.008) | Tag-value text (`:20:`, `:32A:`) |
| **Transport** | HTTPS → EBA STEP2 | SWIFT Alliance Lite2 |
| **Settlement** | T+1 (SCT) / 10s (SCT Inst) | T+1 a T+3 |
| **Respuesta** | Síncrona (pacs.002 en mismo response) | Asíncrona (webhook via SWIFT gpi) |
| **Alcance** | EUR intra-EU | Global |
| **UETR** | Opcional en CT, obligatorio en Inst | Obligatorio (`:121:`) para gpi tracking |

### 7. Double-entry accounting: Debit → Settle/Reverse

```
1. Debit(iban, amount, version)  → fondos bloqueados con guarda OCC
2a. Rail.Send() → ACSC   → Settle(orderID)  → confirma el débito
2b. Rail.Send() → RJCT   → Reverse(orderID) → libera los fondos bloqueados
2c. Rail.Send() → error  → Reverse(orderID) → libera los fondos bloqueados
```

### 8. UETR: el identificador que viaja inmutable

El `UETR` (Unique End-to-End Transaction Reference) es un UUID que el cliente asigna y que **ningún intermediario puede modificar**: viaja en `PaymentOrderInitiated.EventID` (clave de idempotencia en Redis), en `pacs.008` (`PaymentID.UETR`) y en MT103 (`:121:`). Es el mecanismo de trazabilidad end-to-end de SWIFT gpi.

## ISO 20022 — Mensajes implementados

| Mensaje | Nombre | Uso |
|---------|--------|-----|
| `pacs.008.001.08` | FIToFI Customer Credit Transfer | Envío de transferencia a la cámara de compensación |
| `pacs.002.001.10` | FIToFI Payment Status Report | Respuesta de la cámara: `ACSC` (liquidado) o `RJCT` (rechazado) |

## MT103 — Campos clave

| Tag | Nombre | Ejemplo |
|-----|--------|---------|
| `:20:` | Sender's Reference | `INSTR001` (max 16 chars) |
| `:23B:` | Bank Operation Code | `CRED` (siempre para customer transfers) |
| `:32A:` | Value Date + Currency + Amount | `240115EUR1500,00` (coma decimal) |
| `:50K:` | Ordering Customer | `/DE89370400440532013000\nAlice Müller` |
| `:57A:` | Account With Institution | `BNPAFRPP` (BIC del banco receptor) |
| `:59:` | Beneficiary | `/FR76...\nBob Jones` |
| `:71A:` | Details of Charges | `SHA` (shared) · `OUR` (sender) · `BEN` (receiver) |
| `:121:` | UETR | UUID obligatorio para SWIFT gpi tracking |

## Ejecutar y Probar

### 1. Levantar infraestructura local

```bash
cd 10-core-banking-iso20022
docker-compose up -d
docker-compose ps   # esperar que redpanda y redis estén "healthy"
```

Esto levanta:
- **Redpanda** (compatible con protocolo Kafka) en `localhost:9092`
- **Redpanda Console** (UI para inspeccionar el topic) en `http://localhost:8080`
- **Redis** en `localhost:6379`
- Crea el topic `banking.payment.orders.initiated` automáticamente

### 2. Levantar los dos microservicios

```bash
# Terminal 1 — payment-api (HTTP → Kafka producer)
cd payment-api
KAFKA_BROKERS=localhost:9092 go run ./cmd
# → "payment-api listening addr=:8083 brokers=[localhost:9092]"

# Terminal 2 — payment-worker (Kafka consumer → OCC ledger → SEPA/SWIFT)
cd payment-worker
KAFKA_BROKERS=localhost:9092 REDIS_URL=redis://localhost:6379 go run ./cmd
# → "payment-worker consuming from Kafka [localhost:9092]"
```

Dejá ambas terminales abiertas — vas a ver los logs de cada request ahí (útil para confirmar que el worker efectivamente consumió el evento).

### 3. Probar con curl

**Transferencia SEPA** (respuesta rápida, EUR, intra-UE):

```bash
curl -i -X POST http://localhost:8083/payments \
  -H "Content-Type: application/json" \
  -d '{
    "uetr": "550e8400-e29b-41d4-a716-446655440000",
    "end_to_end_id": "E2E001",
    "amount_cents": 150000,
    "currency": "EUR",
    "debtor_iban": "DE89370400440532013000",
    "debtor_bic": "DEUTDEDB",
    "debtor_name": "Alice Müller",
    "creditor_iban": "FR7630004000031234567890143",
    "creditor_bic": "BNPAFRPP",
    "creditor_name": "Bob Jones",
    "rail": "SEPA"
  }'
```

**Transferencia SWIFT** (internacional, procesamiento asíncrono):

```bash
curl -i -X POST http://localhost:8083/payments \
  -H "Content-Type: application/json" \
  -d '{
    "uetr": "660e8400-e29b-41d4-a716-446655440001",
    "end_to_end_id": "E2E002",
    "amount_cents": 500000,
    "currency": "USD",
    "debtor_iban": "GB29NWBK60161331926819",
    "debtor_bic": "NWBKGB2L",
    "debtor_name": "Carlos Ruiz",
    "creditor_iban": "US64SVBKUS6S3300958879",
    "creditor_bic": "SVBKUS6S",
    "creditor_name": "John Smith",
    "rail": "SWIFT"
  }'
```

**Respuesta esperada** (ambos casos, `202 Accepted` inmediato — `payment-api` no espera al banco):

```json
{ "uetr": "550e8400-e29b-41d4-a716-446655440000", "status": "PDNG" }
```

### 4. Verificar que el worker procesó el evento

El `202 Accepted` solo confirma que Kafka recibió el evento — no que el pago se completó. Para confirmar el procesamiento real:

- **`GET /payments/{uetr}`**: la forma correcta de consultar el estado final (ver paso 4.1)
- **Logs de la Terminal 2** (`payment-worker`): deberías ver una línea sin errores tras cada request. Si hay un problema (rail caído, fondos insuficientes), aparece ahí.
- **Redpanda Console** (`http://localhost:8080`) → Topics → `banking.payment.orders.initiated` o `banking.payment.orders.results` → ver los mensajes publicados en crudo.
- **Reenviar el mismo `uetr`**: el segundo POST con el mismo `uetr` prueba la idempotencia — el worker debe loguear `"skipping already-completed order"` en vez de reprocesar.

```bash
# Reenviar el mismo UETR — debe ser un no-op en el worker (idempotencia Redis)
curl -i -X POST http://localhost:8083/payments \
  -H "Content-Type: application/json" \
  -d '{"uetr": "550e8400-e29b-41d4-a716-446655440000", "debtor_iban": "DE89370400440532013000", "creditor_iban": "FR7630004000031234567890143", "amount_cents": 150000, "currency": "EUR", "rail": "SEPA"}'
```

### 4.1. Consultar el estado final del pago

```bash
curl -i http://localhost:8083/payments/550e8400-e29b-41d4-a716-446655440000
```

**Mientras el worker aún no publicó el resultado** (`202 Accepted`):
```json
{ "uetr": "550e8400-e29b-41d4-a716-446655440000", "status": "PDNG" }
```

**Una vez que payment-worker consumió el evento y publicó el resultado** (`200 OK`):
```json
{ "uetr": "550e8400-e29b-41d4-a716-446655440000", "status": "ACSC", "reason": "" }
```

**Si falló por fondos insuficientes** (`200 OK` — el HTTP status es 200 porque la consulta en sí fue exitosa; el estado del pago está en el body):
```json
{ "uetr": "...", "status": "RJCT", "reason": "banking: insufficient balance for debit" }
```

### 4.2. Notificación por webhook (alternativa al polling)

En lugar de hacer polling a `GET /payments/{uetr}`, el cliente puede registrar una URL de callback al momento de iniciar el pago. `payment-api` la invoca automáticamente apenas conoce el resultado final (éxito, rechazo o timeout):

```bash
curl -i -X POST http://localhost:8083/payments \
  -H "Content-Type: application/json" \
  -d '{
    "uetr": "770e8400-e29b-41d4-a716-446655440002",
    "amount_cents": 150000,
    "currency": "EUR",
    "debtor_iban": "DE89370400440532013000",
    "creditor_iban": "FR7630004000031234567890143",
    "rail": "SEPA",
    "callback_url": "https://webhook.site/tu-id-unico"
  }'
```

- La URL se guarda en Redis (`CallbackStore`, TTL = 2× el timeout de negocio) junto al `uetr`.
- Cuando `payment-worker` publica el resultado en `banking.payment.orders.results`, `payment-api` lo consume (`ResultProcessor.HandleResult`), lo persiste, y dispara la notificación en una goroutine separada (no bloquea el consumer de Kafka).
- `WebhookNotifier` hace **POST** a la URL con `{"uetr": "...", "status": "ACSC|RJCT", "reason": "..."}`, con hasta 3 intentos y backoff lineal (1s, 2s), cada intento con timeout de 5s.
- Es "best effort": si el webhook falla las 3 veces, el resultado sigue disponible vía `GET /payments/{uetr}` — el cliente nunca pierde la respuesta, solo el aviso proactivo.
- Si `callback_url` se omite, el flujo sigue siendo puramente por polling (comportamiento anterior, sin cambios).

### 4.3. Timeout de negocio para pagos que nunca resuelven (`TimeoutReaper`)

Si `payment-worker` se cae, un mensaje se pierde, o el rail bancario nunca responde, un pago quedaría en `PDNG` para siempre. `payment-api` corre un `TimeoutReaper` en background que barre periódicamente el índice de pendientes (`PendingIndex`, un ZSET en Redis) y expira automáticamente los que superan el SLA:

- Variable de entorno `PAYMENT_TIMEOUT` (segundos, default `300` = 5 minutos) define el SLA de negocio.
- El *sweep* corre cada 30 segundos (intervalo fijo en `main.go`, ajustable en el código).
- Cada `uetr` detectado como "stale" (más viejo que `PAYMENT_TIMEOUT` y aún sin resultado) se marca como `RJCT` con `reason: "timeout: no result received within business SLA"`, se persiste en `ResultsStore`, se remueve de `PendingIndex`, y dispara el webhook si había uno registrado.
- Esto es una salvaguarda independiente del resultado real del banco — si el resultado legítimo llega después del timeout (mensaje tardío), `payment-worker` ya completó su procesamiento real vía idempotencia; el timeout solo protege al *cliente* de esperar indefinidamente una respuesta que quizás nunca llegue por Kafka.

**Para probar el timeout**: iniciar un pago, matar `payment-worker` (Ctrl+C en la Terminal 2) antes de que procese el evento, y esperar `PAYMENT_TIMEOUT` segundos. `GET /payments/{uetr}` debe pasar de `PDNG` a `RJCT` con el `reason` de timeout, sin que el worker haya hecho nada.

### 4.4. Persistencia (Redis, no en memoria)

`ResultsStore`, `PendingIndex` y `CallbackStore` viven en Redis (mismo `docker-compose.yml`, variable `REDIS_URL` en `payment-api`, default `redis://localhost:6379`) — no en un `map` en memoria. Esto significa que reiniciar `payment-api` no pierde resultados ya conocidos ni pagos pendientes: al reiniciar, el `TimeoutReaper` retoma el barrido con el mismo estado persistido.

### 5. Probar desde Postman

1. Crear un nuevo request `POST` a `http://localhost:8083/payments`
2. En **Headers**: `Content-Type: application/json`
3. En **Body → raw → JSON**: pegar cualquiera de los payloads de arriba
4. Enviar — la respuesta debe llegar en milisegundos con `202 Accepted` y `{"status":"PDNG"}`
5. Cambiar el `uetr` en cada request (es la clave de idempotencia) para simular pagos distintos
6. Crear un segundo request `GET` a `http://localhost:8083/payments/{{uetr}}` (reemplazando `{{uetr}}` por el UUID enviado) para consultar el resultado unos segundos después

### 6. Tests unitarios (sin infraestructura)

```bash
cd banking-core && go test ./... -count=1 -v
```

No requiere Docker ni Kafka — usa `gomock` para simular `Rail` y `Ledger`.


