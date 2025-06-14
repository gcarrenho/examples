### ✅ good-example/ – Arquitectura Hexagonal por Feature, con contratos orientados al consumidor
En este ejemplo mostramos una implementación que sí respeta los principios de la arquitectura hexagonal, aplicada por feature. Esta aproximación presenta varios beneficios clave:

## 🧩 Estructura por feature
En lugar de organizar el código por tipo (handlers, services, repositories), lo organizamos por dominio funcional. Cada feature (por ejemplo, orders, payments, etc.) contiene su propio conjunto de casos de uso, interfaces, modelos y adaptadores.

Esto permite:

- Alta cohesión interna dentro del feature.

- Bajo acoplamiento con otros features.

- Mayor claridad sobre responsabilidades.

- Escalabilidad organizacional (equipos por dominio).

## ✅ Contratos definidos por el consumidor
A diferencia del bad-example, en este diseño:

- orders define una interfaz (contrato) que especifica qué necesita saber de payments.

- Luego payments implementa ese contrato.

Este enfoque se conoce como Dependency Inversion orientada al consumidor: el módulo que necesita la funcionalidad (en este caso, orders) define la interfaz, y el módulo que la puede satisfacer (payments) la implementa.

Esto evita el acoplamiento estructural. Por ejemplo:

```bash
// En orders/core/ports/payment_port.go
type PaymentDTO struct {
	Status string
}

type PaymentStatusChecker interface {
    OrderPaymentStatus(ctx context.Context, orderID string) (PaymentDTO, error)
}
```
- Acá no hay ninguna referencia a domain.Payment ni a detalles internos de payments.

- Si payments cambia su modelo de dominio, orders no se ve afectado siempre que respete el contrato.

- El flujo de dependencia queda claro y controlado.

## 🛡️ Sobre el acceso a los repositorios
Aunque la arquitectura hexagonal por feature guía hacia mejores decisiones, Este tipo de Arquitectura, en conjunto con Golang no impide técnicamente que un desarrollador inyecte directamente un repositorio en un handler o lo use desde otro paquete.

Esto pasa porque:

- En Go, si un tipo o función es exportado (empieza con mayúscula), cualquier paquete puede acceder a él.

- No existen modificadores de acceso como private o protected (como en otros lenguajes).

Por lo tanto:

``` 📌 La arquitectura no te salva sola. Necesitás disciplina, convenciones claras de equipo y una buena revisión de código para evitar mal uso de componentes públicos.```

Una buena práctica en este contexto es:

- Exportar solo interfaces mínimas que representen los contratos necesarios.

- Mantener las implementaciones concretas lo más encapsuladas posible dentro del paquete del feature.

- Nombrar explícitamente los contratos por caso de uso, y evitar interfaces genéricas como Repository o Service.

## 📈 Beneficios de esta aproximación
- ✅ Dominios bien aislados, fáciles de razonar y testear.

- ✅ Evolución independiente de features, sin romper otros.

- ✅ Claridad en las dependencias: siempre se sabe quién depende de quién, y por qué.

- ✅ Escalable para equipos y producto: cada feature puede crecer sin afectar al resto.

- ✅ Fácil mantenimiento a largo plazo.

