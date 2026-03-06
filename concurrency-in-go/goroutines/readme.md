## Gorrutinas

Las gorutinas son una de las unidades de organización más básicas de un programa Go, por lo que es importante que entendamos qué son y cómo funcionan. De hecho, cada programa Go tiene al menos una gorutina: la gorutina `main`, que se crea y se inicia automáticamente cuando comienza el proceso. En casi cualquier programa, probablemente tarde o temprano te encontrarás recurriendo a una gorutina para que te ayude a resolver tus problemas. 

### ¿Qué son?
En términos muy simples, una goroutine es una función que se ejecuta simultáneamente (recuerde: ¡no necesariamente en paralelo!) junto con otro código. Puede iniciar una simplemente colocando la palabra clave go antes de una función:
```go
func main() { 
	go sayHello() 
	// continuar haciendo otras cosas 	
}

func sayHello() { 
	fmt.Println("hello")
}
```

¡Las funciones anónimas también funcionan! Aquí hay un ejemplo que hace lo mismo que el ejemplo anterior; sin embargo, en lugar de crear una goroutine a partir de una función, creamos una goroutine a partir de una función anónima:
```go
go func() { 
	fmt.Println("hello")

}()

// seguir haciendo otras cosas
```
- Tenga en cuenta que invocamos la función anónima inmediatamente despues de la palabra clave `go`.


Alternativamente, puede asignar la función a una variable y llamar a la función anónima de esta manera:
```go
sayHello:= func() { 
	fmt.Println("hello")
}

go sayHello()

// seguir haciendo otras cosas
```

Hay mucho que decir sobre cómo usarlas correctamente, sincronizarlas y organizarlas, pero esto es realmente todo lo que necesitas saber para empezar a utilizarlas.

Las goroutines son exclusivas de Go (aunque otros lenguajes tienen una primitiva de concurrencia similar). No son subprocesos del sistema operativo ni subprocesos (subprocesos que son administrados por el entorno de ejecución de un lenguaje), sino que son un nivel superior de abstracción conocido como corrutinas. Las corrutinas (funciones, cierres o métodos en Go) son simplemente subrutinas concurrentes que no son preventivas, es decir, que no pueden ser interrumpidas. En cambio, las corrutinas tienen múltiples puntos a lo largo de los cuales se puede suspender o reingresar.

Ciertamente es posible tener varias corrutinas ejecutándose secuencialmente para dar la ilusión de paralelismo, y de hecho esto sucede todo el tiempo en Go.


El mecanismo de Go para alojar goroutines es una implementación de lo que se denomina un planificador M:N, lo que significa que asigna M subprocesos verdes a N subprocesos del sistema operativo. Luego, los goroutines se programan en los subprocesos verdes. Cuando tenemos más goroutines que subprocesos verdes disponibles, el planificador maneja la distribución de los goroutines entre los subprocesos disponibles y garantiza que, cuando estos se bloqueen, se puedan ejecutar otros goroutines