// Orquesta components initializing and dependency injection
package app

import (
	"github.com/gcarrenho/component-based/option-1/internal/orders"
	"github.com/gcarrenho/component-based/option-1/internal/payments"
	"github.com/gcarrenho/component-based/option-1/internal/users"
	"github.com/gcarrenho/component-based/option-1/pkg/infra/db"
)

type AppContainer struct {
	OrderComponent   orders.OrderComponent
	PaymentComponent payments.PaymentComponent
	UserComponent    users.UserComponent
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
		UserComponent: users.NewUserComponentImpl(users.Deps{
			DB: db.InitMySQL(),
		}),
	}
}
