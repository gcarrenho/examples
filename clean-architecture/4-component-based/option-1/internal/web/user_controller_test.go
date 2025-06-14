package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gcarrenho/component-based/option-1/internal/users/model"
	"github.com/gcarrenho/component-based/option-1/mocks"
	"github.com/gin-gonic/gin"
	"github.com/huandu/go-assert"
	"go.uber.org/mock/gomock"
)

type mockUserController struct {
	userSvc *mocks.MockUserComponent
}

func TestUserController_getByID(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*mockUserController)
		expectedStatus int
	}{
		{
			name:   "valid request",
			userID: "123",
			mockSetup: func(m *mockUserController) {
				m.userSvc.EXPECT().FindUserByID("123").Return(model.User{
					ID:   "123",
					Name: "Juan",
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "processing error",
			userID: "123",
			mockSetup: func(m *mockUserController) {
				m.userSvc.EXPECT().FindUserByID("123").Return(model.User{}, errors.New("fail"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			m := &mockUserController{
				userSvc: mocks.NewMockUserComponent(mockCtrl),
			}

			tt.mockSetup(m)

			ctrl := NewUserController(m.userSvc)

			r := gin.Default()
			gin.SetMode(gin.TestMode)
			group := r.Group("/user")

			ctrl.RegisterRoutes(group)

			req := httptest.NewRequest(http.MethodGet, "/user/"+tt.userID, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
