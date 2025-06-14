package services

import (
	"context"
	"testing"

	"github.com/gcarrenho/hexagonal/good-example/internal/payments/core/model"
	"github.com/gcarrenho/hexagonal/good-example/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type mockPaymentServices struct {
	paymentRepo *mocks.MockPaymentsRepository
}

func TestFindPaymentByID(t *testing.T) {
	ctx := context.Background()

	expectedPayment := model.Payment{ID: "payment123"}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := mockPaymentServices{
		paymentRepo: mocks.NewMockPaymentsRepository(mockCtrl),
	}

	m.paymentRepo.EXPECT().GetPaymentByID(ctx, "payment123").Return(expectedPayment, nil)
	service := NewPaymentsService(m.paymentRepo)
	payment, err := service.FindPaymentByID(ctx, "payment123")

	assert.NoError(t, err)
	assert.Equal(t, expectedPayment, payment)
}
