package services

import (
	"testing"
)

func TestProcessPayment(t *testing.T) {
	payment := ProcessPayment("order-123")

	if payment.OrderID != "order-123" {
		t.Errorf("expected OrderID 'order-123', got %s", payment.OrderID)
	}

	if payment.Status != "Processed" {
		t.Errorf("expected Status 'Processed', got %s", payment.Status)
	}
}
