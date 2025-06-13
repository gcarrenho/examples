package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/huandu/go-assert"
	"github.com/stretchr/testify/mock"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) FindUserByID(id string) (interface{}, error) {
	args := m.Called(id)
	return args.Get(0), args.Error(1)
}

func TestGetUserByID(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*MockUserService)
		expectedStatus int
	}{
		{
			name:   "success",
			userID: "u1",
			mockSetup: func(m *MockUserService) {
				m.On("FindUserByID", "u1").Return("user123", nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "service error",
			userID: "u2",
			mockSetup: func(m *MockUserService) {
				m.On("FindUserByID", "u2").Return(nil, errors.New("fail"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockService := new(MockUserService)
			tt.mockSetup(mockService)

			ctrl := web.NewUserController(mockService)

			r := gin.Default()
			group := r.Group("/api")
			ctrl.RegisterRoutes(group)

			req := httptest.NewRequest("GET", "/api/user/"+tt.userID, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
