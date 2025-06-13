// Orquesta components initializing and dependency injection
package app

import (
	"github.com/gcarrenho/component-based/option-2/internal/orders"
	"github.com/gcarrenho/component-based/option-2/internal/payments"
	"github.com/gcarrenho/component-based/option-2/pkg/infra/db"
)

type AppContainer struct {
	OrderComponent   orders.OrderComponent
	PaymentComponent payments.PaymentComponent
}

func NewAppContainer() *AppContainer {
	return &AppContainer{
		OrderComponent: orders.NewOrderComponentImpl(orders.Deps{
			DB: db.InitMySQL(),
			//Logger: initZapLogger(),
		}),
		PaymentComponent: payments.NewPaymentComponentImpl(payments.Deps{
			DB: db.InitPostgres(),
			//Logger: initLogrusLogger(),
		}),
	}
}
