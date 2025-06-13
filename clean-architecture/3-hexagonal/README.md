# Arquitectura Hexagonal (Ports and Adapters / Clean Architecture)

La arquitectura hexagonal propone una separación clara entre el núcleo de negocio y los detalles de implementación (infraestructura). El dominio (entidades y casos de uso) debe ser independiente de frameworks, bases de datos, controladores HTTP, etc. Esta idea también es conocida como Clean Architecture o Ports & Adapters.

## 🧵 Estructura típica
```bash
hexagonal/
├── cmd/
│   └── main.go
├── adapters/
└── core/
    ├── models/
    ├── ports/
	└── services/
```
## 🧪 Ventajas

✅ Alta independencia del dominio.  
✅ Fácil testeo del núcleo sin infraestructura.  
✅ Adaptable a cambios tecnológicos (framework, base de datos).  
✅ Claro qué entra y qué sale del sistema.

## ⚠️ Desventajas

⚠️ Mayor complejidad inicial.  
⚠️ Puede ser sobredimensionado para proyectos pequeños.  
⚠️ En proyectos grandes, navegar por ports y adapters se vuelve confuso.
⚠️ Si no se separan bien responsabilidades, termina en una “Layered disfrazada”.  
⚠️ No escala bien en equipos múltiples o dominios separados.

### 📁 `/bad-example`

## ❌ Un diseño común... pero problemático
En este ejemplo, mostramos una implementación típica de arquitectura hexagonal que incurre en varios problemas reales:

- Acoplamiento estructural

- Repositorios públicos y mal ubicados.

- Arquitectura en capas encubierta.

Esto lleva a un diseño que es rígido, difícil de navegar y con bajo nivel de cohesión.

### 📁 `/good-example`

## ✅ Estructura por Feature, con contratos claros
Este ejemplo presenta una arquitectura hexagonal por feature, lo cual ofrece varias ventajas clave:

- Altísima cohesión.

- Evita acoplamiento estructural.

- Contratos explícitos entre features.

- Escalabilidad por equipos.


## 💡 Conclusión
La arquitectura hexagonal es poderosa, pero mal implementada puede traer más problemas que soluciones. Adoptar una estructura por feature, con contratos claros entre dominios, y evitando el acoplamiento estructural, permite escalar tanto técnica como organizacionalmente.

No se trata solo de usar interfaces o separar infraestructura: se trata de diseñar software guiado por el negocio, no por la tecnología.