# Concurrencia en Go

Este repositorio, titulado `currency-in-go`, está dedicado a explorar los **conceptos de concurrencia** y los **patrones en Go**. Su objetivo es proporcionar una comprensión integral de cómo Go maneja la concurrencia, inspirado en los principios de los **Procesos Secuenciales Comunicantes (CSP)** y en la filosofía del lenguaje Go.

---

## Descripción General

La concurrencia en Go no se trata de paralelismo, sino de estructurar aplicaciones para manejar múltiples tareas simultáneamente. Este repositorio explica:
- Conceptos fundamentales de concurrencia en Go.
- Patrones comunes de concurrencia.
- Mejores prácticas para escribir programas concurrentes eficientes y seguros.

---

## Conceptos Clave

1. **Goroutines**  
   Las goroutines son hilos ligeros gestionados por el runtime de Go, lo que permite ejecutar tareas concurrentes de manera eficiente.

2. **Canales**  
   Los canales facilitan la comunicación y sincronización entre goroutines de forma segura, siguiendo el mantra de Go:  
   *“No compartas memoria para comunicarte; comunica para compartir memoria.”*

3. **Primitivas de Sincronización**  
   Herramientas como `sync.Mutex` y `sync.WaitGroup` proporcionan control de bajo nivel para casos donde los canales no son ideales.

4. **Concurrencia vs Paralelismo**  
   - Concurrencia: Manejo de múltiples tareas al mismo tiempo.  
   - Paralelismo: Ejecución simultánea de múltiples tareas.

---

## Cómo Utilizar Este Repositorio

- Explora los ejemplos de código proporcionados para aprender sobre los conceptos y patrones de concurrencia.
- Utiliza estos patrones como base para diseñar sistemas concurrentes escalables y eficientes.
- Experimenta con los ejemplos y modifícalos para adaptarlos a tus casos de uso específicos.

---

## Recursos

- [Documentación de Go](https://golang.org/doc/)
- [Concurrency in Go de Katherine Cox-Buday](https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/)

---

## Contribuir

¡Siéntete libre de contribuir con ejemplos adicionales, explicaciones o mejoras a los patrones existentes!