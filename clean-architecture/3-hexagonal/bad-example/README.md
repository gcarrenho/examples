### ❌ bad-example/ – Un diseño común... pero problemático
Este ejemplo representa una implementación que parece seguir los principios de la arquitectura hexagonal, pero en la práctica introduce varios errores frecuentes:

## 🔗 Acoplamiento estructural
Uno de los problemas más serios es el acoplamiento estructural entre features. Por ejemplo:

```bash
// Orders need to know if a payment is successful or not.
// So payments define a method to check the status of a payment for an order.
// But this is not the responsibility of the payment service.
OrderPaymentStatus(ctx context.Context, orderID string) (domain.Payment, error)
```
En este caso:

- El servicio de pagos (payments) expone una función que devuelve una entidad interna (domain.Payment) para que el servicio de órdenes (orders) pueda consultar su estado.

- El problema es que orders conoce y depende de la estructura interna de payments, lo que genera un fuerte acoplamiento estructural.

- Si cambia la implementación o el modelo del dominio de payments, orders se rompe, porque no está acoplado a un comportamiento, sino a una estructura.

- Además, se viola la dirección de la dependencia: orders es quien debería definir qué necesita, y payments debería implementar ese contrato.

Este tipo de diseño:

- Dificulta la evolución independiente de los features.

- Aumenta el riesgo de errores al modificar el modelo de un dominio.

- Provoca diseños frágiles y difíciles de mantener.

## 🛑 Repositorios públicos y acceso directo
Otro problema frecuente es que los repositorios se declaran públicos y están disponibles para cualquiera:

- En este ejemplo, el handler puede recibir directamente un repositorio en lugar de pasar por un caso de uso o servicio de aplicación.

- Aunque en teoría la interfaz protege, en la práctica nada impide que alguien use el repositorio directamente si está expuesto.

- Esto es especialmente peligroso para quienes son nuevos en la arquitectura, ya que rompe la separación entre capas y genera confusión.

## 🧱 Arquitectura en capas disfrazada
Aunque esté organizada como "hexagonal", esta estructura termina funcionando como una arquitectura por capas tradicional:

- handlers, services, repositories... todos crecen horizontalmente, sin foco ni cohesión por feature.

- No hay una guía clara de límites de dominio.

- A medida que crece el proyecto, se vuelve difícil navegarlo y entender qué hace cada componente.