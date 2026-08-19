> 🌐 [English](README.md)

# Caso 02 — Control de Concurrencia Optimista vs Pesimista

## El Problema: Actualización Perdida

Dos goroutines debitan la misma cuenta simultáneamente:

```
Goroutine A lee:    saldo=$100, versión=5
Goroutine B lee:    saldo=$100, versión=5
Goroutine A escribe: saldo=$40, versión=6   (debitó $60)
Goroutine B escribe: saldo=$40, versión=6   (debitó $60, sobreescribe la escritura de A)

Resultado: saldo=$40 — pero se aplicaron dos débitos de $60. La cuenta perdió $60.
```

Esta es la anomalía **Lost Update** (Actualización Perdida). Ambos escritores leyeron
datos obsoletos y el segundo sobreescribió al primero sin saber que hubo un commit.

## Control de Concurrencia Optimista (OCC)

Optimismo: "las colisiones son raras; que todos intenten y se detecten conflictos al escribir."

```sql
-- 1. Leer con versión
SELECT balance, version FROM accounts WHERE id = $1;

-- 2. Escribir con guarda de versión
UPDATE accounts
   SET balance = $new_balance, version = version + 1
 WHERE id = $1 AND version = $expected_version;

-- Si 0 filas se actualizaron → versión no coincide → otro escritor hizo commit → reintentar
```

**Ideal para**: contención baja a media. Las lecturas no bloquean; los conflictos son raros.

## Control de Concurrencia Pesimista (PCC)

Pesimismo: "las colisiones son costosas; adquirir un lock exclusivo antes de leer."

```sql
BEGIN;
-- Lock a nivel de fila: ninguna otra transacción puede actualizar esta fila hasta el COMMIT
SELECT balance FROM accounts WHERE id = $1 FOR UPDATE;
UPDATE accounts SET balance = balance + $delta WHERE id = $1;
COMMIT;
```

**Ideal para**: alta contención. Los escritores hacen cola en lugar de reintentar en bucle.

## Comparación OCC vs PCC

| Dimensión | OCC | PCC |
|-----------|-----|-----|
| Modelo de concurrencia | Falla rápido + reintento | Bloquea y espera |
| Rendimiento (baja contención) | ✅ Mayor | ❌ Sobrecarga del lock |
| Rendimiento (alta contención) | ❌ Muchos reintentos | ✅ Predecible |
| Riesgo de deadlock | Ninguno | Posible |
| Complejidad de implementación | Bucle de reintento | Gestión de transacciones |

## ¿Por Qué PostgreSQL para OCC/PCC?

PostgreSQL provee los mejores bloques de construcción nativos para ambas estrategias en un solo motor.

| Base de datos | OCC | PCC | Notas |
|---------------|-----|-----|-------|
| **PostgreSQL** | ✅ `UPDATE … WHERE version=$v` | ✅ `SELECT FOR UPDATE` | MVCC: los lectores nunca bloquean escritores. El lock a nivel de fila es preciso. `UPDATE … RETURNING` entrega el nuevo estado de forma atómica. |
| **MySQL / MariaDB** | ✅ columna versión funciona | ⚠️ gap locks | `SELECT FOR UPDATE` sobre un rango puede adquirir gap locks entre filas, generando deadlocks inesperados en PCC con consultas de rango. |
| **MongoDB** | ✅ `findOneAndUpdate` + `$where` | ⚠️ solo nivel documento | Las transacciones multi-documento se agregaron en v4.0 pero tienen sobrecarga de sesión. OCC es idiomático; PCC requiere two-phase locking manual. |
| **CockroachDB** | ✅ serializable por defecto | ✅ `SELECT FOR UPDATE` | Compatible con Postgres pero distribuido: cada round-trip de lock/conflicto cruza la red — mayor latencia p99 que Postgres en un solo nodo. |
| **DynamoDB** | ✅ `ConditionExpression: version=:v` | ❌ sin lock de fila | Las escrituras condicionales son compatibles con OCC pero no existe equivalente a `SELECT FOR UPDATE`. PCC debe emularse con ítems de lock separados (complejo y propenso a errores). |
| **Redis** | ⚠️ `WATCH / MULTI / EXEC` | ❌ sin lock de fila | Las transacciones optimistas con WATCH existen pero están limitadas a operaciones de clave simples. No está diseñado para datos financieros relacionales. Usar Redis para claves de idempotencia (Caso 01), no para gestión de saldos. |
| **SQLite** | ✅ sí | ✅ sí | Modelo de único escritor; la base de datos misma es el lock. No apto para sistemas distribuidos pero útil para desarrollo local y tests. |

**Por qué Postgres gana específicamente en pagos:**

1. **MVCC** — Multi-Version Concurrency Control: `SELECT` nunca bloquea `UPDATE`, por lo que las cargas de trabajo OCC con muchas lecturas no matan de inanición a los escritores.
2. **Serializable Snapshot Isolation (SSI)** — Postgres puede detectar y abortar automáticamente anomalías de serialización sin ninguna columna `version` (con `isolation level serializable`).
3. **Cláusula `RETURNING`** — `UPDATE accounts SET balance=$1, version=version+1 WHERE id=$2 AND version=$3 RETURNING balance` lee el nuevo estado confirmado en el mismo round-trip, eliminando un segundo `SELECT`.
4. **Advisory locks** — `pg_advisory_xact_lock(hashtext(account_id))` provee locks nombrados a nivel de aplicación para PCC sin tocar la fila en absoluto.
5. **Índices parciales** — `CREATE INDEX ON payments (account_id) WHERE status = 'PENDING'` hace que la consulta de guarda de versión sea O(log n) sobre el subconjunto activo de filas.

## Ejecutar los Tests

```bash
go test ./02-concurrency-occ/... -race -v -count=1
```

Observá la línea de log `OCC conflicts (retries)`. Con 100 goroutines, OCC produce
típicamente 400–900 conflictos para 100 escrituras exitosas — cada conflicto es trabajo
desperdiciado. PCC produce cero conflictos porque el mutex serializa a todos los escritores.

## ¿Qué Pasa Si el Pod se Cae Durante la Operación?

**OCC**: el pod lee la cuenta (saldo + versión) y luego se cae antes de ejecutar `UpdateOCC`.
No se escribió ningún estado — la fila de la base de datos queda completamente intacta.
La siguiente solicitud lee datos frescos y procede normalmente. OCC es naturalmente
seguro ante crashes porque la escritura es un único `UPDATE` atómico; no hay estado
parcial intermedio.

**PCC**: el pod llama a `SELECT FOR UPDATE` (adquiere el lock de fila) y luego se cae
antes del `COMMIT`. PostgreSQL detecta la conexión TCP rota y **hace rollback automático
de la transacción y libera el lock de fila**. No se necesita ningún código de aplicación
para la recuperación.

```
Pod A: SELECT FOR UPDATE → 💥 crash
           │
           │ Conexión TCP cerrada → Postgres hace rollback + libera lock (segundos)
           ▼
Pod B: SELECT FOR UPDATE → UPDATE → COMMIT ✓
```

**Conclusión clave**: OCC no deja ningún estado en la base de datos al caerse; la
recuperación es instantánea. PCC mantiene el lock hasta que la base de datos detecta
la conexión muerta. Configurar `tcp_keepalives_idle` en el pool de conexiones de pgx
para detectar crashes en segundos en lugar del default del SO (varios minutos).
