package services

import (
	"testing"

	"github.com/gcarrenho/hexagonal/good-example/internal/orders/core/model"
	"github.com/gcarrenho/hexagonal/good-example/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type mockOrdersServices struct {
	orderRepo *mocks.MockOrdersRepository
}

func TestFindOrderByID(t *testing.T) {
	expectedOrder := model.Order{
		ID:     "order123",
		Status: "Pending",
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := mockOrdersServices{
		orderRepo: mocks.NewMockOrdersRepository(mockCtrl),
	}

	m.orderRepo.EXPECT().GetOrderByID("order123").Return(expectedOrder, nil)

	service := NewOrdersService(m.orderRepo)
	order, err := service.FindOrderByID("order123")

	assert.NoError(t, err)
	assert.Equal(t, expectedOrder, order)
}
