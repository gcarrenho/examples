> 🌐 [English](README.md)

# Caso 04 — sync.Pool vs Asignación en el Heap

## El Problema: Presión sobre el GC a Escala

Construir un string canónico de pago (para claves de idempotencia y firma HMAC) en
cada solicitud entrante asigna un buffer temporal `[]byte`:

```
Llamada 1: alloc []byte → construir string → string(b) → GC recolecta el []byte
Llamada 2: alloc []byte → construir string → string(b) → GC recolecta el []byte
...
```

A 1 millón de solicitudes por segundo, el GC recolecta millones de buffers de corta
vida por segundo. Esto aumenta la frecuencia de las pausas stop-the-world del GC e
infla la latencia en el percentil 99 — exactamente donde se producen las violaciones de SLA.

## La Solución: sync.Pool

`sync.Pool` mantiene una caché de objetos reutilizables por P (por hilo del SO).

- **`Get()`** devuelve un buffer reciclado del pool local del P — **sin asignación**.
- **`Put()`** devuelve el buffer para que lo use el próximo llamador en el mismo P.
- El robo entre Ps es automático; el pool escala linealmente con `GOMAXPROCS`.

```go
var pool = sync.Pool{New: func() any { b := make([]byte, 0, 256); return &b }}

bp := pool.Get().(*[]byte)
b := (*bp)[:0]      // ← resetear el largo, preservar la capacidad — siempre hacer esto
defer func() {
    *bp = b         // escribir de vuelta por si append hizo crecer el slice
    pool.Put(bp)
}()
```

## Invariante Crítico: Siempre Resetear Antes de Usar

El pool puede devolver un buffer con datos de un llamador anterior. Olvidar `b = b[:0]`
es un **bug silencioso de corrupción de datos** — no va a hacer panic, va a anteponer
bytes obsoletos que son casi imposibles de detectar en producción.

## Resultados Esperados del Benchmark

```
BenchmarkCanonical_WithPool-8          20_000_000     58 ns/op    48 B/op    1 allocs/op
BenchmarkCanonical_Direct-8             8_000_000    145 ns/op   160 B/op    5 allocs/op
BenchmarkCanonical_Parallel_WithPool-8 60_000_000     20 ns/op    48 B/op    1 allocs/op
```

El `1 alloc/op` restante en `WithPool` es la copia del `string(b)` al final — el
llamador es dueño de ese string, por lo que no puede reciclarse. El `[]byte` de scratch
se elimina por completo del heap.

## El `1 alloc/op` que No Puede Eliminarse

```
Pool.Get()   → []byte reciclado del pool (sin alloc)
append(...)  → construir el string en el buffer reciclado (sin alloc)
string(b)    → UNA copia al heap — el llamador es dueño de este string ← aquí está el 1 alloc
Pool.Put()   → devolver el []byte al pool (sin alloc)
```

Para llegar a **0 allocs/op**, habría que devolver `[]byte` en lugar de `string`, pero
eso requiere que el llamador gestione el tiempo de vida del slice, lo cual complica la API.
El trade-off vale la pena: 1 alloc vs 5 allocs es una reducción del 80%.

## Ejecutar los Benchmarks

```bash
# Benchmark básico con estadísticas de asignaciones
go test ./04-memory-syncpool/... -bench=. -benchmem -count=3

# Observar el comportamiento del Pool con distintos valores de GOMAXPROCS (debería escalar linealmente)
go test ./04-memory-syncpool/... -bench=Parallel -benchmem -cpu=1,2,4,8
```

## ¿Qué Pasa Si el Pod se Cae Durante la Operación?

`sync.Pool` es **exclusivamente memoria de scratch en proceso** — no guarda ningún estado
de negocio, ni dinero, ni saldos de cuentas. Un crash del pod descarta el pool junto con
todo el proceso, y eso es completamente seguro.

Si un pod se cae mientras construye un string canónico de pago, el request HTTP falla
a nivel de transporte. El cliente reintenta. El siguiente pod arranca con un pool frío
(todos los buffers recien asignados desde el heap, igual que al inicio), que es el estado
normal de arranque.

El filtro de idempotencia del Caso 01 asegura que el reintento no sea procesado dos veces.

**Impacto de un crash en sync.Pool: ninguno.** La única consideración es el breve pico
de latencia después de un reinicio mientras el pool se calienta (las primeras N llamadas
asignan buffers frescos del heap antes de que el pool tenga objetos reciclados para ofrecer).
