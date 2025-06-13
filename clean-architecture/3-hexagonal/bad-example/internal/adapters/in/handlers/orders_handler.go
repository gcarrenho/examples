package handlers

import (
	"net/http"

	"github.com/gcarrenho/hexagonal/bad-example/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type OrdersHandler struct {
	orderService ports.OrdersService
}

func NewOrdersHandler(rg *gin.RouterGroup, orderService ports.OrdersService) {
	orderHandler := &OrdersHandler{
		orderService: orderService}
	rg.GET("/orders/:id", orderHandler.getByID)
}

func (h *OrdersHandler) getByID(c *gin.Context) {
	orderID := c.Param("id")
	order, err := h.orderService.FindOrderByID(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order"})
		return
	}
	c.JSON(http.StatusOK, order)
}
