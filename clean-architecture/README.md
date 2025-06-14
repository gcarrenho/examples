# 🏗️ Clean Architecture Examples in Go

Este repositorio contiene ejemplos educativos de cómo aplicar distintos enfoques de arquitectura de software en Go (Golang), basados en el concepto de *Clean Architecture* propuesto por Robert C. Martin (Uncle Bob).

Su objetivo es mostrar, comparar y entender los beneficios y compromisos de cada enfoque, manteniendo un diseño desacoplado, testable y mantenible.

---

## 🧭 Estructuras disponibles

Cada subdirectorio representa un enfoque diferente de arquitectura:

- `1-package-by-layered`: Arquitectura clásica en capas (handlers → services → repositories).
- `2-package-by-feature`: Agrupamiento por dominio o "feature" (ej: `users`, `orders`, etc.).
- `3-hexagonal`: Arquitectura Hexagonal (Ports and Adapters).
- `4-component-based`: Arquitectura por Componentes (división modular, encapsulada y autónoma).

---

## 🎓 Objetivo educativo

Cada ejemplo:
- Sigue principios SOLID.
- Muestra separación de responsabilidades.
- Se enfoca en cómo estructurar el código, no en la lógica de negocio.

---

## ⚠️ ¿Qué no incluye?

- No está pensado para producción.
- No incluye autenticación, middlewares avanzados ni manejo de errores exhaustivo.
- Usa librerías conocidas de forma mínima.

---

## 🧵 Organización sugerida

```bash
/
├── 1-package-by-layered/
├── 2-package-by-feature-based/
├── 3-hexagonal/
└── 4-component-based/
    ├── option-1/
    └── option-2/
```

## 🧵 Contribuciones

¡Pull requests y sugerencias son bienvenidos!


## 📜 Licencia

MIT