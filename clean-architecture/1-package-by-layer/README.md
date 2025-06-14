# 📚 Package By Layer Architecture

Este enfoque organiza la aplicación por capas técnicas:

- `handlers` o `controllers`: reciben la petición y delegan la lógica.
- `services`: contienen la lógica de negocio.
- `repositories`: abstraen el acceso a datos.

---

## 🧪 Ventajas

✅ Simple y fácil de entender.  
✅ Común en muchas aplicaciones tradicionales.  
✅ Separación clara entre capas.

---

## ⚠️ Desventajas

⚠️ Puede escalar mal en sistemas grandes.  
⚠️ Puede generar acoplamiento entre capas técnicas.  
⚠️ Dificulta el trabajo paralelo en equipos grandes.  
⚠️ La lógica del dominio queda perdida entre detalles técnicos.

---

## 🧵 Estructura

```bash
handlers/
services/
repositories/
models/
```

## 🎓 Ideal para

- Aplicaciones pequeñas o medianas.

- Equipos que recién comienzan con arquitectura limpia.