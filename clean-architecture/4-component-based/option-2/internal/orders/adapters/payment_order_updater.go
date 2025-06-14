package adapters

import (
	"github.com/gcarrenho/component-based/option-2/internal/orders"
	"github.com/gcarrenho/component-based/option-2/internal/payments/contracts/order"
)

var _ order.OrderUpdater = (*PaymentOrderUpdater)(nil)

type PaymentOrderUpdater struct {
	orderSvc *orders.OrderComponent
}

// MarkOrderAsPaid implements order.OrderUpdater.
func (o *PaymentOrderUpdater) MarkOrderAsPaid(orderID string, paymentID string) error {
	// o.orderSvc.UpdateOrder()
	return nil
}
