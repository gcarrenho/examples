package routes

import (
	"github.com/gcarrenho/component-based/option-2/cmd/app"
	"github.com/gcarrenho/component-based/option-2/internal/web"
	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine, container *app.AppContainer) {
	api := router.Group("/api")
	/*
		private := api.Group("/private")
		private.Use(JWTMiddleware())

		admin := api.Group("/admin")
		admin.Use(AdminMiddleware())
	*/
	// Public (sin auth)
	//public := api.Group("/public")
	// public.Use(...)

	// Private (con auth middleware)
	private := api.Group("/private")
	//private.Use(AuthMiddleware())

	// Payments
	paymentController := web.NewPaymentController(container.PaymentComponent)
	paymentController.RegisterRoutes(private.Group("/payments"))

	// Orders
	//orderController := orders.NewController( /* ... */ )
	//orderController.RegisterRoutes(private.Group("/orders"))

	// Health
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
