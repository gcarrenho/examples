// Adaptadores HTTP
package web

import (
	"fmt"
	"net/http"

	"github.com/gcarrenho/component-based/option-1/internal/payments"
	"github.com/gin-gonic/gin"
)

type PaymentController struct {
	service payments.PaymentComponent // Interface for payment processing
}

func NewPaymentController(service payments.PaymentComponent) *PaymentController {
	return &PaymentController{service: service}
}

func (c *PaymentController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/process", c.processPayment)
}

func (c *PaymentController) processPayment(ctx *gin.Context) {
	var req struct {
		OrderID string  `json:"order_id"`
		Amount  float64 `json:"amount"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	p, err := c.service.ProcessPayment(req.OrderID, req.Amount)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "processing failed"})
		return
	}
	fmt.Println("Payment processed successfully:", p)

	ctx.JSON(http.StatusCreated, gin.H{"status": "success"})
}
