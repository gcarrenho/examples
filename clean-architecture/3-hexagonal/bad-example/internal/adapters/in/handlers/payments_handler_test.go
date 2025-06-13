package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gcarrenho/hexagonal/bad-example/internal/core/model"
	"github.com/gcarrenho/hexagonal/bad-example/mocks"
	"github.com/gin-gonic/gin"
	"github.com/huandu/go-assert"
	"go.uber.org/mock/gomock"
)

type mockPaymentsHandler struct {
	paymentService *mocks.MockPaymentsService
}

func TestGetPaymentByID(t *testing.T) {

	type want struct {
		status int
		error  error
	}

	type testCase struct {
		name    string
		orderID string
		mocks   func(m mockPaymentsHandler)
		want    want
	}

	tests := []testCase{
		{
			name:    "success - order found",
			orderID: "123",
			want:    want{status: http.StatusOK, error: nil},
			mocks: func(m mockPaymentsHandler) {
				m.paymentService.EXPECT().
					FindPaymentByID(context.Background(), "123").
					Return(model.Payment{ID: "123", Status: "PENDING"}, nil)
			},
		},
		{
			name:    "failure - order service error",
			orderID: "123",
			want:    want{status: http.StatusInternalServerError, error: errors.New("db error")},
			mocks: func(m mockPaymentsHandler) {
				m.paymentService.EXPECT().FindPaymentByID(context.Background(), "123").Return(model.Payment{}, errors.New("db error"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := gin.Default()
			gin.SetMode(gin.TestMode)
			rg := r.Group("")

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			m := mockPaymentsHandler{
				paymentService: mocks.NewMockPaymentsService(mockCtrl),
			}
			tc.mocks(m)

			NewPaymentsHandler(rg, m.paymentService)

			req, _ := http.NewRequest(http.MethodGet, "/payments/"+tc.orderID, nil)

			r.ServeHTTP(w, req)

			assert.Equal(t, tc.want.status, w.Code)
		})
	}
}
