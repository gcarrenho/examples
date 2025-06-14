# Arquitectura Basada en Características (Feature-Based Architecture)

Este enfoque organiza el código según las `funcionalidades del dominio`, en lugar de por capas técnicas. Cada `feature` (o funcionalidad) contiene todo lo necesario para funcionar de manera aislada: controladores, casos de uso, modelos, etc.

## 🧵 Estructura típica

```bash
feature-based/
├── users/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   └── model.go
├── orders/
│   ├── handler.go
│   ├── service.go
│   └── model.go
├── shared/
│   └── db.go
└── main.go
```

## 🧪 Ventajas
✅ Refleja el dominio real del negocio.  
✅ Fomenta equipos alineados por feature.  
✅ Alto nivel de cohesión dentro de cada feature.  
✅ Más fácil de escalar con nuevas funcionalidades.   

## ⚠️ Desventajas

⚠️ Riesgo de duplicación si no se gestionan bien los elementos compartidos.  
⚠️ No siempre está claro qué responsabilidades deben vivir dentro de cada feature.  
⚠️ Puede haber inconsistencias entre features si no hay buenas prácticas comunes.  
⚠️ Si todo está “adentro” del handler, el acoplamiento se esconde.

