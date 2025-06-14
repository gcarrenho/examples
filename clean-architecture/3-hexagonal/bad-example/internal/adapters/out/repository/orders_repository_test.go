package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOrderByID(t *testing.T) {
	tests := []struct {
		name     string
		orderID  string
		expected string
	}{
		{
			name:     "valid order ID",
			orderID:  "abc123",
			expected: "abc123",
		},
		{
			name:     "another order ID",
			orderID:  "xyz789",
			expected: "xyz789",
		},
	}

	repo := NewOrdersRepository()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := repo.GetOrderByID(tt.orderID)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, order.ID)
			assert.Equal(t, "Pending", order.Status)
		})
	}
}
