package adapter

import (
	"errors"
	"testing"

	"github.com/gcarrenho/component-based/option-1/internal/users/model"
	"github.com/gcarrenho/component-based/option-1/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type paymentAdapterMock struct {
	userService *mocks.MockUserComponent
}

func TestIsUserActive(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		mockSetup   func(m *paymentAdapterMock)
		expectError bool
		expected    string
	}{
		{
			name:   "user found",
			userID: "u1",
			mockSetup: func(m *paymentAdapterMock) {
				m.userService.EXPECT().FindUserByID("u1").Return(model.User{
					ID:   "u1",
					Name: "Alice",
				}, nil)
			},
			expected: "u1",
		},
		{
			name:   "user not found returns error",
			userID: "u404",
			mockSetup: func(m *paymentAdapterMock) {
				m.userService.EXPECT().FindUserByID("u404").Return(model.User{}, errors.New("not found"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			m := &paymentAdapterMock{
				userService: mocks.NewMockUserComponent(mockCtrl),
			}
			tt.mockSetup(m)

			adapter := NewPaymentAdapter(m.userService)

			res, err := adapter.IsUserActive(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, res.ID)
			}
		})
	}
}
