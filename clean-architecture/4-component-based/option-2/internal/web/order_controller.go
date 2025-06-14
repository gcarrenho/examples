package web

import (
	"net/http"

	"github.com/gcarrenho/component-based/option-2/internal/orders"
	"github.com/gin-gonic/gin"
)

type OrderController struct {
	service orders.OrderComponent // Interface for payment processing
}

func NewOrderController(service orders.OrderComponent) *OrderController {
	return &OrderController{service: service}
}

func (c *OrderController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/:id", c.getByID)
}

func (c *OrderController) getByID(ctx *gin.Context) {
	orderID := ctx.Param("id")
	order, err := c.service.FindOrderByID(orderID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"order": order})
}
