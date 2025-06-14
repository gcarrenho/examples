package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gcarrenho/component-based/option-1/internal/orders/model"
	"github.com/gcarrenho/component-based/option-1/mocks"
	"github.com/gin-gonic/gin"
	"github.com/huandu/go-assert"
	"go.uber.org/mock/gomock"
)

type mockOrderController struct {
	orderSvc *mocks.MockOrderComponent
}

func TestOrderController_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string
		mockSetup      func(*mockOrderController)
		expectedStatus int
	}{
		{
			name:    "valid request",
			orderID: "123",
			mockSetup: func(m *mockOrderController) {
				m.orderSvc.EXPECT().FindOrderByID("123").Return(model.Order{
					ID:     "123",
					Status: "Completed",
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "processing error",
			orderID: "123",
			mockSetup: func(m *mockOrderController) {
				m.orderSvc.EXPECT().FindOrderByID("123").Return(model.Order{}, errors.New("fail"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := gin.Default()

			gin.SetMode(gin.TestMode)
			rg := r.Group("/orders")

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			m := &mockOrderController{
				orderSvc: mocks.NewMockOrderComponent(mockCtrl),
			}
			tc.mockSetup(m)

			ctr := NewOrderController(m.orderSvc)
			ctr.RegisterRoutes(rg)

			req, _ := http.NewRequest(http.MethodGet, "/orders/"+tc.orderID, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
