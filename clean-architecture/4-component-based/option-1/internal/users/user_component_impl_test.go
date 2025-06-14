package users

import (
	"errors"
	"testing"

	"github.com/gcarrenho/component-based/option-1/internal/users/model"
	"github.com/gcarrenho/component-based/option-1/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestFindUserByID(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		mockSetup   func(m *mocks.MockuserRepository)
		expectedID  model.User
		expectError bool
	}{
		{
			name:   "user found",
			userID: "123",
			mockSetup: func(m *mocks.MockuserRepository) {
				m.EXPECT().GetUserByID("123").Return(model.User{
					ID: "123",
				}, nil)
			},
			expectedID: model.User{
				ID: "123",
			},
		},
		{
			name:   "user not found",
			userID: "404",
			mockSetup: func(m *mocks.MockuserRepository) {
				m.EXPECT().GetUserByID("404").Return(model.User{}, errors.New("not found"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockuserRepository(ctrl)
			tt.mockSetup(mockRepo)

			comp := &UserComponentImpl{
				repo: mockRepo,
			}

			id, err := comp.FindUserByID(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}
