package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPaymentByID(t *testing.T) {
	tests := []struct {
		name        string
		paymentID   string
		expectedID  string
		expectedErr bool
	}{
		{
			name:        "returns static payment ID",
			paymentID:   "anyvalue",
			expectedID:  "12345", // hardcoded in repo
			expectedErr: false,
		},
	}

	repo := NewPaymentsRepository()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := repo.GetPaymentByID(ctx, tt.paymentID)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, payment.ID)
			}
		})
	}
}
