package handlers

import (
	"net/http"

	"github.com/gcarrenho/hexagonal/good-example/internal/payments/core/ports"

	"github.com/gin-gonic/gin"
)

type PaymentsHandler struct {
	paymentService ports.PaymentsService
}

func NewPaymentsHandler(rg *gin.RouterGroup, paymentService ports.PaymentsService) {
	paymentHandler := &PaymentsHandler{
		paymentService: paymentService}
	rg.GET("/payments/:id", paymentHandler.getByID)
}

func (p *PaymentsHandler) getByID(c *gin.Context) {
	paymentID := c.Param("id")
	order, err := p.paymentService.FindPaymentByID(c.Request.Context(), paymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve payment"})
		return
	}
	c.JSON(http.StatusOK, order)
}
