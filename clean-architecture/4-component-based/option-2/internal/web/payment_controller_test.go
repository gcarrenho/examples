package web

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gcarrenho/component-based/option-2/internal/payments/mocks"
	"github.com/gin-gonic/gin"
	"github.com/huandu/go-assert"
	"go.uber.org/mock/gomock"
)

type mockPaymentsHandler struct {
	paymentService *mocks.MockPaymentComponent
}

func TestHandleProcessPayment(t *testing.T) {
	tests := []struct {
		name           string
		payload        string
		mockSetup      func(*mockPaymentsHandler)
		expectedStatus int
	}{
		{
			name:    "valid request",
			payload: `{"order_id":"123", "amount":100.5}`,
			mockSetup: func(m *mockPaymentsHandler) {
				m.paymentService.EXPECT().ProcessPayment("123", 100.5).Return("ok", nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid json",
			payload:        `{"order_id":123, "amount":"abc"}`,
			mockSetup:      func(m *mockPaymentsHandler) {}, // No call expected
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "processing error",
			payload: `{"order_id":"123", "amount":100.5}`,
			mockSetup: func(m *mockPaymentsHandler) {
				m.paymentService.EXPECT().ProcessPayment("123", 100.5).Return("", errors.New("fail"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			m := &mockPaymentsHandler{
				paymentService: mocks.NewMockPaymentComponent(mockCtrl),
			}

			tt.mockSetup(m)

			ctrl := NewPaymentController(m.paymentService)

			r := gin.Default()
			gin.SetMode(gin.TestMode)
			group := r.Group("/payment")

			ctrl.RegisterRoutes(group)

			req := httptest.NewRequest(http.MethodPost, "/payment/process", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
