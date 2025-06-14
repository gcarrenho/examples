package handlers

import (
	"net/http"

	"github.com/gcarrenho/hexagonal/bad-example/internal/core/ports"

	"github.com/gin-gonic/gin"
)

// PaymentsHandlerBad is a handler for payments that directly interacts with the repository.
// This is an anti-pattern as it violates the separation of concerns and makes the handler dependent on the repository.
// Ideally, the handler should interact with a service layer that encapsulates the business logic.
// This handler is used to demonstrate the bad practice of directly accessing the repository from the handler.
// But it very easy to fall into this trap, especially in small projects or when developers are not familiar with the principles of clean architecture.
// Because in hexagonal architecture, the repository needs to be public to be used by the services because they live in different packages.
// And the compiler wouldn't tell us anything about it because it's fine, we're accessing something that we can access because it's public
type PaymentsHandlerBad struct {
	rg          *gin.RouterGroup
	paymentRepo ports.PaymentsRepository
}

func NewPaymentsHandlerBad(rg *gin.RouterGroup, paymentRepo ports.PaymentsRepository) {
	paymentHandler := &PaymentsHandlerBad{
		paymentRepo: paymentRepo}
	rg.GET("bad/payments/:id", paymentHandler.getByID)
}

func (p *PaymentsHandlerBad) getByID(c *gin.Context) {
	paymentID := c.Param("id")
	order, err := p.paymentRepo.GetPaymentByID(c.Request.Context(), paymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve payment"})
		return
	}
	c.JSON(http.StatusOK, order)
}
