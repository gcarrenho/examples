package orders

import (
	"errors"
	"testing"

	"github.com/gcarrenho/component-based/option-1/internal/orders/model"
	"github.com/gcarrenho/component-based/option-1/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type orderComponentMock struct {
	orderRepo *mocks.MockorderRepository
}

func TestFindOrderByID(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		mockSetup   func(repo orderComponentMock)
		expected    model.Order
		expectError bool
	}{
		{
			name:    "order found successfully",
			orderID: "123",
			mockSetup: func(m orderComponentMock) {
				m.orderRepo.EXPECT().GetOrderByID("123").Return(model.Order{
					ID:     "123",
					Status: "Confirmed",
				}, nil)
			},
			expected: model.Order{ID: "123", Status: "Confirmed"},
		},
		{
			name:    "order not found returns error",
			orderID: "not-found",
			mockSetup: func(m orderComponentMock) {
				m.orderRepo.EXPECT().GetOrderByID("not-found").Return(model.Order{}, errors.New("order not found"))

			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			m := orderComponentMock{
				orderRepo: mocks.NewMockorderRepository(mockCtrl),
			}
			tt.mockSetup(m)

			comp := &OrderComponentImpl{orderRepo: m.orderRepo}

			order, err := comp.FindOrderByID(tt.orderID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, order)
			}

		})
	}
}
