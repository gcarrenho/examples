package order

import "testing"

func TestCreateOrder(t *testing.T) {
	order := CreateOrder("order-1", 99.99)

	if order.ID != "order-1" {
		t.Errorf("expected ID 'order-1', got %s", order.ID)
	}

	if order.Amount != 99.99 {
		t.Errorf("expected Amount 99.99, got %f", order.Amount)
	}
}
