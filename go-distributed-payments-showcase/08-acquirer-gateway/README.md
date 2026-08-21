# Case 08 — Acquirer Gateway: Strategy Pattern + Circuit Breaker

## El sistema

Un único servicio que enruta autorizaciones de tarjeta al adquirente correcto según el país de la tarjeta. Cada adquirente tiene su propio protocolo propietario; el servicio absorbe esa varianza.

```
POST /acquire
      │
   Router (Strategy)
      │
      ├── "AR" → Prisma   → HTTP → api.prisma.com/autorizacion
      ├── "BR" → Cielo    → HTTP → api.cielo.com.br/1/sales
      └── "*"  → Adyen    → HTTP → checkout-test.adyen.com/v68/payments
```

## Decisiones de diseño

### 1. Un servicio único con Strategy, no uno por adquirente

La tentación es crear `acquirer-ar-svc`, `acquirer-br-svc`, `acquirer-eu-svc`. El problema: el circuit breaker, el logging de autorización, el retry con idempotencia y la traducción de request son lógica **idéntica** para los tres. Cuatro servicios = cuatro lugares donde romperla.

Un único servicio con el patrón Strategy: el `Router` selecciona el `Gateway` correcto, cada `Gateway` traduce el formato propietario. La lógica compartida vive una sola vez.

**Cuándo sí partir en servicios distintos:**
- Regulación: data residency diferente por país (GDPR/Europa vs LGPD/Brasil)
- Equipos distintos que necesitan deploys independientes
- Un país procesa 100x más que otro y necesita escalar solo

### 2. `Gateway` exportado — puerto de extensión intencional

A diferencia de los repositorios (que son privados), `Gateway` es exportado porque `cmd/main.go` necesita componer los adaptadores en el `Router`.

```go
// internal/gateway.go
type Gateway interface {
    Authorize(ctx context.Context, req AuthRequest) (AuthResponse, error)
}

// cmd/main.go — cmd puede nombrar Gateway porque es la capa de composición
ar := gateway.WithCircuitBreaker(prisma.New(...), 5)
eu := gateway.WithCircuitBreaker(adyen.New(...), 5)
router := gateway.NewRouter(eu, map[string]gateway.Gateway{"AR": ar})
```

Contraste con `payment-svc` donde `idempotencyStore` y `accountRepository` son privados — nadie fuera del package puede inyectarlos directamente.

### 3. Circuit breaker por adquirente, no global

Cada adquirente tiene su propio circuit breaker independiente:

```go
ar := gateway.WithCircuitBreaker(prisma.New(...), threshold)  // breaker de Prisma
eu := gateway.WithCircuitBreaker(adyen.New(...), threshold)   // breaker de Adyen
```

Si Prisma (Argentina) cae, el breaker de AR se abre. Las tarjetas europeas siguen siendo autorizadas por Adyen sin interrupción. Un breaker global afectaría a todos aunque solo un adquirente esté caído.

### 4. Los adaptadores traducen `AuthRequest` ↔ formato propietario

El servicio habla `AuthRequest`/`AuthResponse` (tipos del package). Cada adaptador absorbe la varianza de protocolo internamente:

```go
// internal/prisma/gateway.go — nadie ve estos tipos fuera del package
type authorizeRequest struct {
    NumeroTransaccion string  `json:"nro_transaccion"` // Prisma usa español
    Monto             float64 `json:"monto"`           // pesos, no centavos
}

// internal/adyen/gateway.go — completamente distinto
type paymentRequest struct {
    Reference string `json:"reference"`
    Amount    amount `json:"amount"` // Adyen usa centavos directamente
}
```

El servicio y el handler nunca ven estos tipos. Si Adyen cambia su API, solo cambia `internal/adyen/gateway.go`.

### 5. Estructura flat dentro de `internal/`

```
08-acquirer-gateway/internal/
├── gateway.go     ← AuthRequest, AuthResponse, errores, Gateway interface, Router, CircuitBreaker
├── handler.go     ← adaptador HTTP de entrada (mismo package)
├── prisma/        ← sub-package: depende de json propietario de Prisma
│   └── gateway.go
└── adyen/         ← sub-package: depende de json propietario de Adyen
    └── gateway.go
```

`gateway.go` y `handler.go` son `package gateway`. Los adaptadores son sub-packages porque tienen dependencias de protocolo distintas y se pueden testear de forma independiente contra servidores fake.

## Ejecutar

```bash
cd 08-acquirer-gateway && go run ./cmd

# Tarjeta argentina → Prisma
curl -X POST http://localhost:8082/acquire \
  -H "Content-Type: application/json" \
  -d '{"idempotency_key":"txn-001","amount_cents":5000,"currency":"ARS","card_bin":"411111","card_country":"AR","merchant_id":"MERCH001"}'

# Tarjeta europea → Adyen (fallback)
curl -X POST http://localhost:8082/acquire \
  -H "Content-Type: application/json" \
  -d '{"idempotency_key":"txn-002","amount_cents":5000,"currency":"EUR","card_bin":"401288","card_country":"DE","merchant_id":"MERCH001"}'
```
