package services

import (
	"testing"
)

func TestCreateOrder(t *testing.T) {
	order := CreateOrder("test-id", 50.0)

	if order.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %s", order.ID)
	}

	if order.Amount != 50.0 {
		t.Errorf("expected Amount 50.0, got %f", order.Amount)
	}
}
