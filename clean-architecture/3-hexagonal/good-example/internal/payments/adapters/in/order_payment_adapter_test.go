package adapter

import (
	"context"
	"testing"

	"github.com/gcarrenho/hexagonal/good-example/internal/orders/core/ports"
	"github.com/gcarrenho/hexagonal/good-example/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type orderPaymentAdapterMock struct {
	paymentSvc *mocks.MockPaymentsService
}

func TestOrderPaymentStatus(t *testing.T) {
	tests := []struct {
		name      string
		orderID   string
		mockSetup func(mock orderPaymentAdapterMock)
		want      ports.PaymentDTO
		expectErr bool
	}{
		{
			name:    "returns empty DTO without error",
			orderID: "order123",
			mockSetup: func(mock orderPaymentAdapterMock) {
				// No behavior defined because actual implementation is empty
			},
			want: ports.PaymentDTO{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			m := orderPaymentAdapterMock{
				paymentSvc: mocks.NewMockPaymentsService(mockCtrl),
			}

			tt.mockSetup(m)

			adapter := NewOrderPayment(m.paymentSvc)

			got, err := adapter.OrderPaymentStatus(context.Background(), tt.orderID)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			//mockPaymentService.AssertExpectations(t)
		})
	}
}
