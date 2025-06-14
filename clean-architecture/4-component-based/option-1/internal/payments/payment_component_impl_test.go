package payments

import (
	"errors"
	"testing"

	"github.com/gcarrenho/component-based/option-1/internal/payments/model"
	"github.com/gcarrenho/component-based/option-1/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestProcessPayment(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		mockSetup   func(m *mocks.MockpaymentRepository)
		expectedID  string
		expectError bool
	}{
		{
			name:    "payment found",
			orderID: "123",
			mockSetup: func(m *mocks.MockpaymentRepository) {
				m.EXPECT().GetPaymentByID("123").Return(model.Payment{
					ID: "123",
				}, nil)
			},
			expectedID: "123",
		},
		{
			name:    "payment not found",
			orderID: "404",
			mockSetup: func(m *mocks.MockpaymentRepository) {
				m.EXPECT().GetPaymentByID("404").Return(model.Payment{}, errors.New("not found"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockpaymentRepository(ctrl)
			tt.mockSetup(mockRepo)

			comp := &PaymentComponentImpl{
				paymentRepository: mockRepo,
			}

			id, err := comp.ProcessPayment(tt.orderID, 100.0)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}
