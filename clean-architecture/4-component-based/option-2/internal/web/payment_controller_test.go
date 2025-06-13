package web

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/huandu/go-assert"
	"github.com/stretchr/testify/mock"
)

type MockPaymentService struct {
	mock.Mock
}

func (m *MockPaymentService) ProcessPayment(orderID string, amount float64) (interface{}, error) {
	args := m.Called(orderID, amount)
	return args.Get(0), args.Error(1)
}

func TestHandleProcessPayment(t *testing.T) {
	tests := []struct {
		name           string
		payload        string
		mockSetup      func(*MockPaymentService)
		expectedStatus int
	}{
		{
			name:    "valid request",
			payload: `{"order_id":"123", "amount":100.5}`,
			mockSetup: func(m *MockPaymentService) {
				m.On("ProcessPayment", "123", 100.5).Return("payment123", nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid json",
			payload:        `{"order_id":123, "amount":"abc"}`,
			mockSetup:      func(m *MockPaymentService) {}, // No call expected
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "processing error",
			payload: `{"order_id":"123", "amount":100.5}`,
			mockSetup: func(m *MockPaymentService) {
				m.On("ProcessPayment", "123", 100.5).Return(nil, errors.New("fail"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockService := new(MockPaymentService)
			tt.mockSetup(mockService)

			ctrl := web.NewController(mockService)

			r := gin.Default()
			group := r.Group("/payment")
			ctrl.RegisterRoutes(group)

			req := httptest.NewRequest("POST", "/payment/process", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
