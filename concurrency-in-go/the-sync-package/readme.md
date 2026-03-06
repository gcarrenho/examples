# El paquete de sincronización (`sync`)

El paquete `sync` contiene primitivas de concurrencia diseñadas para la sincronización de acceso a memoria de bajo nivel. Si has trabajado en otros lenguajes que manejan concurrencia principalmente mediante sincronización de acceso a memoria, probablemente ya estés familiarizado con estos conceptos. 

La diferencia clave en Go es que combina estas primitivas con un modelo de concurrencia basado en goroutines y canales, proporcionando un conjunto más amplio de herramientas para manejar la concurrencia de manera eficiente. Como se menciona en "La filosofía de Go sobre la concurrencia", estas primitivas son especialmente útiles en ámbitos pequeños, como dentro de una estructura o una sección crítica del código.

A continuación, exploraremos algunas de las principales herramientas que expone el paquete `sync`:

## `sync.WaitGroup`

`WaitGroup` es una herramienta excelente para esperar a que se complete un conjunto de operaciones concurrentes cuando:

1. No necesitas preocuparte por los resultados de esas operaciones, o
2. Tienes otros medios para recopilar dichos resultados.

Si ninguna de estas condiciones aplica, es mejor usar canales y una declaración `select`. A continuación, un ejemplo básico de cómo usar un `WaitGroup` para esperar a que se completen varias goroutines:

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	sayHello := func() {
		defer wg.Done()
		fmt.Println("hello")
	}

	wg.Add(1)
	go sayHello()
	wg.Wait() // Este es el punto de unión (join point)
}
```

## `sync.Mutex` y `sync.RWMutex`
Si estás familiarizado con lenguajes que manejan concurrencia mediante sincronización de acceso a memoria, reconocerás inmediatamente Mutex.

### ¿Qué es un Mutex?
Un Mutex (exclusión mutua) protege secciones críticas de tu programa, que son áreas que requieren acceso exclusivo a recursos compartidos.

Mientras que los canales en Go comparten memoria comunicándose, un Mutex comparte memoria mediante una convención que sincroniza el acceso. Es tu responsabilidad coordinar este acceso, asegurándote de que cualquier código que utilice un recurso compartido lo proteja con un Mutex.

Ejemplo básico de uso de sync.Mutex para sincronizar acceso:
```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var mu sync.Mutex
	var value int

	increment := func() {
		mu.Lock()
		defer mu.Unlock()
		value++
	}

	decrement := func() {
		mu.Lock()
		defer mu.Unlock()
		value--
	}

	increment()
	decrement()

	fmt.Println(value)
}
```

## `sync.Once`
El tipo sync.Once garantiza que un fragmento de código se ejecute solo una vez, sin importar cuántas goroutines intenten ejecutarlo simultáneamente. Esto es útil para inicializaciones que deben ejecutarse exactamente una vez, como la configuración de una conexión global o el registro de un logger.

Ejemplo básico de uso de sync.Once:
```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var once sync.Once
	initialize := func() {
		fmt.Println("Inicializado solo una vez")
	}

	for i := 0; i < 3; i++ {
		go func() {
			once.Do(initialize)
		}()
	}

	// Permitir que las goroutines se ejecuten
	fmt.Scanln()
}
```

## `sync.Cond`
El tipo sync.Cond actúa como un punto de encuentro para goroutines que necesitan esperar o anunciar la ocurrencia de un evento.

### ¿Qué es un evento?
Un "evento" en este contexto es una señal arbitraria entre goroutines que no lleva información adicional, solo que ha ocurrido.

Ejemplo básico usando un bucle infinito para esperar señales (un enfoque ingenuo):
```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	cond := sync.NewCond(&sync.Mutex{})
	ready := false

	go func() {
		time.Sleep(2 * time.Second)
		cond.L.Lock()
		ready = true
		cond.Signal() // Notifica a una goroutine que está esperando
		cond.L.Unlock()
	}()

	cond.L.Lock()
	for !ready {
		cond.Wait() // Espera la señal
	}
	cond.L.Unlock()

	fmt.Println("Evento recibido, continuando ejecución")
}
```

# `sync.Pool`
sync.Pool es una implementación segura y concurrente del patrón de pool de objetos. Aunque la explicación completa de este patrón puede encontrarse en la literatura sobre patrones de diseño, aquí ofrecemos una visión general de por qué podría interesarte usarlo.

### ¿Qué es un Pool?
Un pool es una forma de crear y poner a disposición una cantidad fija, o pool, de objetos para su uso. Este patrón se utiliza comúnmente para restringir la creación de cosas que son costosas, como conexiones de bases de datos. Así, solo se crea una cantidad fija de ellas, pero múltiples operaciones pueden acceder a estos recursos de manera concurrente.

En Go, el tipo `sync.Pool` puede ser utilizado de manera segura por múltiples goroutines.

### ¿Cómo funciona?
El método principal de `sync.Pool` es `Get`. Cuando lo llamas:

 1. Primero verifica si hay instancias disponibles dentro del pool.
 2. Si no hay instancias disponibles, llama a la variable miembro New para crear una nueva.
 3. Cuando termines de usar el objeto, puedes devolverlo al pool usando el método Put.

Ejemplo básico:
```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	pool := sync.Pool{
		New: func() interface{} {
			return "Nuevo objeto"
		},
	}

	// Obtener un objeto del pool
	obj := pool.Get()
	fmt.Println(obj)

	// Devolver un objeto al pool
	pool.Put("Objeto reutilizado")
	fmt.Println(pool.Get())
}
```

Este archivo ofrece una visión general y ejemplos prácticos de cómo utilizar las primitivas del paquete `sync`. Cada sección incluye un ejemplo simple y explicaciones claras para facilitar la comprensión y la implementación en proyectos reales.