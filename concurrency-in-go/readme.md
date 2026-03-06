## La Concurrencia es Difícil:
La concurrencia se refiere a la ejecución de múltiples tareas de forma simultánea o intercalada dentro de un sistema de procesamiento. Aunque la concurrencia puede mejorar el rendimiento y la eficiencia, manejar múltiples tareas concurrentes puede ser difícil por varias razones:

- **Condiciones de carrera:** Ocurren cuando dos o más operaciones deben ejecutarse en el orden correcto, pero el programa no ha sido escrito para garantizar que ese orden se mantiene.
Esto mayormente se manifiesta donde una operacion concurrente intenta leer una variable mientras que en un momento indeterminado otra operacion concurrente intenta escribir en la misma variable.
	- **Ejemplo:** Aqui tanto en la linea 3 como en la 5 estan tratando de acceder a la misma variable pero no haya garantia del orden en que se hara.
	```go
	1 var data int
	2 go func(){
	3	data++
	4 }()
	5 if data == 0 {
	6 	fmt.Println("the value is %v. ", data)
	7 }
	
- **Atomicity:** Algo se considera atomico, cuando dentro del contexto en el que opera es indivisible o ininterrumpible.
La atomicidad de una operacion puede cambiar dependiendo del alcance definido actualmente.
	- **Ejemplo:**
	```go
		i++
	```
	Puede parecer atomico, pero un analisis breve revela varias operaciones:
	 - Recuperar el valor de i.
	 - Incrementar el valor de i.
	 - Almacenar el valor de i.

	Si bien cada una de estas operaciones es atomica, la combiancion de todas ellas, puede no serlo, depende del contexto.
	Si su contexto es una go-rutina que no expone i a otras gorutinas entonces este codigo es atomico
- **Sincronización:** Cuando múltiples tareas acceden a recursos compartidos, se requiere sincronización para evitar que las tareas interfieran entre sí, lo que puede causar errores, inconsistencias y fallos en el programa.

	Un claro ejemplo de esto, es nuestro ejemplo de condicion de carrera donde con algunas modificaciones(sinronizacion) podria funcionar correctamente.

	Sin embargo existe un nombre para la seccionde codigo que necesita acceso exclusivo "seccion creitica"

- **Deadlocks, Livelocks y Starvation:** En situaciones donde las tareas dependen unas de otras, pueden producirse bloqueos o "deadlocks" si no se gestionan correctamente.
	- **Deadloks:** Un programa entra en deadlock cuando todoss los procesos concurrentes esperan uno del otro. En este caso el programa nuinca se recuperara sin intervencion externa.
	- **Livelocks:** Son programas que realizan activamente operaciones concurrentes, pero que no hacen nada para avanzar el estado del programa.
		```go
		func main() {
			cadence := sync.NewCond(&sync.Mutex{})
			go func() {
				for range time.Tick(1 * time.Millisecond) {
					cadence.Broadcast()
				}
			}()

			takeStep := func() {
				cadence.L.Lock()
				cadence.Wait()
				cadence.L.Unlock()
			}

			tryDir := func(dirName string, dir *int32, out *bytes.Buffer) bool { // <1>
				fmt.Fprintf(out, " %v", dirName)
				atomic.AddInt32(dir, 1) // <2>
				takeStep()              // <3>
				if atomic.LoadInt32(dir) == 1 {
					fmt.Fprint(out, ". Success!")
					return true
				}
				takeStep()
				atomic.AddInt32(dir, -1) // <4>
				return false
			}

			var left, right int32
			tryLeft := func(out *bytes.Buffer) bool { return tryDir("left", &left, out) }
			tryRight := func(out *bytes.Buffer) bool { return tryDir("right", &right, out) }
			walk := func(walking *sync.WaitGroup, name string) {
				var out bytes.Buffer
				defer func() { fmt.Println(out.String()) }()
				defer walking.Done()
				fmt.Fprintf(&out, "%v is trying to scoot:", name)
				for i := 0; i < 5; i++ { // <1>
					if tryLeft(&out) || tryRight(&out) { // <2>
						return
					}
				}
				fmt.Fprintf(&out, "\n%v tosses her hands up in exasperation!", name)
			}

			var peopleInHallway sync.WaitGroup // <3>
			peopleInHallway.Add(2)
			go walk(&peopleInHallway, "Alice")
			go walk(&peopleInHallway, "Barbara")
			peopleInHallway.Wait()
		} 
		```
	- **Starvation:** Situacion en la que un procesos concurrente no puede obtener todos los recursos que necesita para realizar un trabajo.
		```go
			func main() {
			var wg sync.WaitGroup
			var sharedLock sync.Mutex
			const runtime = 1 * time.Second

			greedyWorker := func() {
				defer wg.Done()

				var count int
				for begin := time.Now(); time.Since(begin) <= runtime; {
					sharedLock.Lock()
					time.Sleep(3 * time.Nanosecond)
					sharedLock.Unlock()
					count++
				}

				fmt.Printf("Greedy worker was able to execute %v work loops\n", count)
			}

			politeWorker := func() {
				defer wg.Done()

				var count int
				for begin := time.Now(); time.Since(begin) <= runtime; {
					sharedLock.Lock()
					time.Sleep(1 * time.Nanosecond)
					sharedLock.Unlock()

					sharedLock.Lock()
					time.Sleep(1 * time.Nanosecond)
					sharedLock.Unlock()

					sharedLock.Lock()
					time.Sleep(1 * time.Nanosecond)
					sharedLock.Unlock()

					count++
				}

				fmt.Printf("Polite worker was able to execute %v work loops.\n", count)
			}

			wg.Add(2)
			go greedyWorker()
			go politeWorker()

			wg.Wait()
		}
		```
	- **Complejidad:** El código concurrente es difícil de escribir, leer y depurar debido a la interacción compleja entre tareas concurrentes.

# Concurrency Patterns and Philosophy in Go

## ¿Estás intentando transferir la propiedad de los datos?

- **Concepto:** Si tienes datos que deben ser utilizados por otro fragmento de código, estás transfiriendo la propiedad de esos datos. 
  - Similar a cómo funciona la propiedad de memoria en lenguajes como C o Rust.
  - Solo una goroutine debe tener la propiedad de los datos en un momento dado.

- **Solución en Go:**
  - Usa **canales** para transferir datos entre goroutines de forma segura.
  - Los canales codifican la intención de transferir propiedad en su diseño, asegurando que una sola goroutine sea responsable de los datos a la vez.

- **Ventajas:**
  - Los canales con buffer desacoplan productores de consumidores.
  - Facilitan que el código concurrente sea componible con otros sistemas concurrentes.

---

## ¿Estás intentando proteger el estado interno de una estructura?

- **Contexto:** Si múltiples goroutines necesitan leer o escribir sobre el mismo recurso compartido, debes proteger ese estado.
  - Ejemplo: Un contador que es incrementado desde varias goroutines.

- **Solución en Go:**
  - Usa primitivas de sincronización como `sync.Mutex`.
  - Esto asegura que solo una goroutine acceda al recurso a la vez.

- **Ejemplo:**
  ```go
  type Counter struct {
      mu    sync.Mutex
      value int
  }

  func (c *Counter) Increment() {
      c.mu.Lock()
      defer c.mu.Unlock()
      c.value++
  }

- **Clave del diseño:**

	Mantén las secciones protegidas (bloqueadas) pequeñas y específicas.
	Esto evita bloqueos prolongados o complicaciones cuando otros fragmentos de código necesitan acceder al mismo recurso.

---

## ¿Estás intentando coordinar múltiples piezas de lógica?
- **Contexto:**  Si tienes varias goroutines que necesitan coordinarse (por ejemplo, esperar a que todas terminen o intercambiar información entre ellas).
	- **Ejemplo:** Procesar tareas en paralelo y recoger los resultados.

- **Solución en Go:**
	- Canales son la mejor opción aquí porque son fáciles de componer.
	- Go ofrece la instrucción select, que permite escuchar varios canales al mismo tiempo, lo que hace que la coordinación sea más sencilla.

 - **Ventajas de los canales:**
	- Puedes usarlos como colas para desacoplar productores y consumidores.
	- Facilitan la depuración porque es más claro cómo se mueven los datos.
	- Los canales son naturalmente seguros para su uso concurrente.

---

## ¿Es una sección crítica para el rendimiento?
- **Contexto:** Si has identificado un cuello de botella importante (después de perfilar tu código), el uso de canales podría ser más lento que otras alternativas.
Esto se debe a que los canales internamente también utilizan mecanismos de sincronización.

- **Solución en Go:**
	- Considera usar primitivas como sync.Mutex o variables atómicas (sync/atomic) para optimizar estas áreas críticas.
	- Sin embargo, si necesitas optimizar algo a este nivel, también deberías considerar reestructurar tu programa.

---

## ¿Qué patrón usar?
- **Usa canales cuando:**
	- Necesites transferir datos entre goroutines.
	- Quieras coordinar múltiples piezas de lógica concurrente.
- **Usa primitivas de sincronización (como Mutex):**
	- Cuando necesitas proteger el acceso a un estado compartido.

---
## ¿Qué pasa con las goroutines?
Trátalas como un recurso barato. Puedes iniciar muchas goroutines sin preocuparte demasiado por el hardware, ya que Go las gestiona de manera eficiente.
Si encuentras problemas de rendimiento relacionados con la concurrencia, probablemente el diseño de tu programa necesite ajustes.

## Filosofía de Go sobre la concurrencia
- **Simplicidad:** Prefiere soluciones claras y fáciles de entender.
- **Canales cuando sea posible:** Usarlos facilita el diseño y reduce errores.
- **Goroutines como un recurso gratuito:** No temas usarlas; Go está diseñado para manejarlas eficientemente.
- **Evita patrones tradicionales como "thread pools":** Go los reemplaza eficientemente con goroutines.