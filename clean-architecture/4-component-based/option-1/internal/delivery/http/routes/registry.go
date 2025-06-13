// Orquesta controladores
package routes

import (
	"github.com/gcarrenho/component-based/option-1/cmd/app"
	"github.com/gcarrenho/component-based/option-1/internal/web"
	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine, container *app.AppContainer) {
	api := router.Group("/api")

	// Public (sin auth)
	public := api.Group("/public")

	// Private (con auth middleware)
	private := api.Group("/private")
	//private.Use(AuthMiddleware())

	admin := api.Group("/admin")
	//admin.Use(AdminMiddleware())

	// Payments
	paymentController := web.NewPaymentController(container.PaymentComponent)
	paymentController.RegisterRoutes(private.Group("/payments"))

	// Orders
	orderController := web.NewOrderController(container.OrderComponent)
	orderController.RegisterRoutes(public.Group("/orders"))

	// Users
	userController := web.NewUserController(container.UserComponent)
	userController.RegisterRoutes(admin.Group("/users"))

	// Health
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
