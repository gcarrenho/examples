package payment

import "testing"

func TestProcessPayment(t *testing.T) {
	payment := ProcessPayment("order-xyz")

	if payment.OrderID != "order-xyz" {
		t.Errorf("expected OrderID 'order-xyz', got %s", payment.OrderID)
	}

	if payment.Status != "Processed" {
		t.Errorf("expected Status 'Processed', got %s", payment.Status)
	}
}
