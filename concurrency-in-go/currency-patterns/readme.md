## Patrones de Concurrencia
Cuando trabajamos con  codigo concurrente existen algunas opciones diferentes para una operacion segura.
- Primitivas de sincronizacion para compartir memoria(mutexs).
- Sincronizacion a partir de la comiunicacion(canales).

Este repositorio incluye implementaciones prácticas de los siguientes patrones:

1. **Confinamiento**
	- Ad-hoc:
	- Lexico:
2. **Fan-out/Fan-in**  
   Distribuir eficientemente tareas a múltiples trabajadores y agregar resultados.

3. **Pipeline**  
   Crear una serie de etapas donde cada etapa procesa datos y los pasa a la siguiente.

4. **Pool de Trabajadores**  
   Gestionar un número fijo de trabajadores procesando trabajos concurrentemente.

5. **Select Statements**  
   Manejar múltiples canales para la comunicación entre goroutines.

---