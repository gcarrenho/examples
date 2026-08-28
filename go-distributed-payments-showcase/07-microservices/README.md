> 🌐 [English](README.md) · [Español](#)

# Case 07 — Microservicios: orders-svc · payment-svc · fraud-svc

## El sistema

Tres servicios que se comunican exclusivamente por HTTP. No comparten código Go entre sí — el único contrato compartido es el API HTTP.

```
Cliente HTTP
     │
     ├──▶  orders-svc  :8081   POST /orders
     │          │
     │          └── HTTP ──▶  payment-svc  :8080   POST /payments/charge
     │
     └──▶  fraud-svc   :8082   POST /fraud/process
                │
                ├── HTTP ──▶  payment-svc  :8080   POST /payments/reserve
                └── HTTP ──▶  payment-svc  :8080   POST /payments/charge
```

## Decisiones de diseño

### 1. El consumidor define su propio contrato

Ningún servicio importa las interfaces de Go del proveedor. Cada consumidor declara exactamente lo que necesita.

```go
// orders-svc/internal/orders.go — solo Charge
type charger interface {
    Charge(ctx, key, accountID string, amountCents int64) error
}

// fraud-svc/internal/deps.go — Charge + Reserve
type paymentGateway interface {
    Charge(ctx, key, accountID string, amountCents int64) error
    Reserve(ctx, key, accountID string, amountCents int64) (reservationID string, err error)
}
```

`payment-svc` no sabe que ninguna de estas interfaces existe. Expone capacidades; cada consumidor toma el subconjunto que necesita. Si mañana aparece `billing-svc` que solo necesita `Balance`, define `type balanceProvider interface { Balance(...) }` sin tocar `payment-svc`.

**Por qué no importar la interfaz del proveedor:** acopla el ciclo de deploy. Si `payment-svc` cambia su interfaz exportada, fuerza recompilación de todos los consumidores aunque no cambió ningún método que usen.

---

### 2. Un bounded context = un package = `internal/`

Cada servicio tiene un único package principal dentro de `internal/`. No hay sub-packages por capa (`domain/`, `service/`, `repository/`).

```
payment-svc/internal/
├── payment.go    ← tipos, errores, puertos privados, struct, New()
├── charge.go     ← métodos Charge y Refund
├── reserve.go    ← método Reserve
├── handler.go    ← adaptador HTTP de entrada
└── memory/       ← sub-package solo porque tiene dependencias de protocolo distintas
    └── store.go
```

Todos los archivos en `internal/` son `package payment`. `charge.go` puede llamar a `acquireKey` definido en `charge.go` y usado por `reserve.go` sin importaciones extras. Go ya provee la modularidad: un archivo por concern dentro del mismo package.

**Por qué no `domain/` + `service/` + `adapters/`:** son nombres de capas arquitectónicas, no de contenido. `package domain` no dice nada sobre qué hay adentro. `package payment` sí.

---

### 3. Interfaces privadas para puertos de salida

Los puertos secundarios (lo que el servicio necesita de infraestructura) son **unexported**. Solo `New()` puede recibir implementaciones.

```go
// payment-svc/internal/payment.go
type idempotencyStore interface { ... }   // minúscula → solo este package la ve
type accountRepository interface { ... }  // minúscula → idem
type reservationStore interface { ... }   // minúscula → idem

func New(idem idempotencyStore, repo accountRepository, rsv reservationStore) *Service
```

`cmd/main.go` llama `payment.New(memStore, memRepo, memRsv)` y Go verifica estructuralmente que los tipos satisfacen las interfaces — sin que `cmd` pueda nombrar ni referenciar las interfaces privadas.

**Por qué:** impide que un junior inyecte el repositorio directamente en el handler. El compilador lo detecta; no hace falta una revisión de PR.

---

### 4. `deps.go` para las dependencias externas (no `ports.go`)

Cuando `fraud.go` tiene más de 2-3 interfaces mezcladas con lógica de negocio, se separan a un archivo dedicado.

```
fraud-svc/internal/
├── fraud.go    ← lógica de negocio pura
├── deps.go     ← interfaces de dependencias externas del package
├── handler.go
└── payment/
    └── client.go
```

El nombre es `deps.go`, no `ports.go`. `ports` es terminología de Hexagonal Architecture (Alistair Cockburn). En Go, los archivos se nombran por lo que contienen, no por el patrón que representan.

| ❌ Terminología de capas | ✅ Idiomático Go |
|---|---|
| `ports.go` | `deps.go` |
| `adapters.go` | `client.go`, `server.go` |
| `domain.go` | `payment.go`, `order.go` |
| `repository.go` | `store.go`, `db.go` |
| `usecases.go` | `service.go` o el nombre del concepto |

---

### 5. Reserve como capacidad explícita de payment-svc

`fraud-svc` usa un flujo de dos fases: reservar fondos → scoring → cobrar si aprobado.

```
fraud-svc:  Reserve("rsv:key", acc, 5000)  →  payment-svc crea hold
            score(accountID, amountCents)   →  fraude aprobado
            Charge("key", acc, 5000)        →  payment-svc debita
```

Esto permite que `fraud-svc` confirme la disponibilidad de fondos **antes** de gastar tiempo en el modelo de fraude. Si los fondos no existen, el request falla rápido sin consumir recursos de scoring.

`orders-svc` no necesita `Reserve` — hace Charge directo porque no tiene paso de fraude en el flujo. Dos consumidores, dos interfaces distintas, mismo proveedor.

---

### 6. Crecimiento dentro del package, no en sub-packages

Cuando un archivo supera ~300 líneas, la respuesta en Go es **más archivos en el mismo package**, no más packages.

```
# payment.go creció → se parte horizontalmente sin nuevas dependencias
payment-svc/internal/
├── payment.go   ← tipos + errores + puertos + struct + New()   ~70 líneas
├── charge.go    ← Charge, Refund, acquireKey, applyDelta       ~65 líneas
├── reserve.go   ← Reserve, newReservationID                    ~40 líneas
└── handler.go   ← HTTP handler                                 ~70 líneas
```

Todos siguen siendo `package payment`. `reserve.go` llama a `acquireKey` definido en `charge.go` directamente — sin imports, sin indirección.

**La señal de que hay que partir el servicio (no el archivo):** cuando distintos equipos necesitan desplegar capacidades de forma independiente. En ese punto se crean nuevos microservicios, no nuevos sub-packages.

---

## Ejecutar los servicios

```bash
# Terminal 1
cd payment-svc && go run ./cmd

# Terminal 2
cd orders-svc && go run ./cmd

# Terminal 3
cd fraud-svc && go run ./cmd

# Probar orders-svc
curl -X POST http://localhost:8081/orders \
  -H "Content-Type: application/json" \
  -d '{"account_id":"acc-alice","amount_cents":1000}'

# Probar fraud-svc (aprobado — < $5000)
curl -X POST http://localhost:8082/fraud/process \
  -H "Content-Type: application/json" \
  -d '{"idempotency_key":"txn-001","account_id":"acc-alice","amount_cents":100000}'

# Probar fraud-svc (bloqueado — > $5000)
curl -X POST http://localhost:8082/fraud/process \
  -H "Content-Type: application/json" \
  -d '{"idempotency_key":"txn-002","account_id":"acc-alice","amount_cents":600000}'
```
