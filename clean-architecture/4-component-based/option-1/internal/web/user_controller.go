package web

import (
	"net/http"

	"github.com/gcarrenho/component-based/option-1/internal/users"
	"github.com/gin-gonic/gin"
)

type UserController struct {
	service users.UserComponent
}

func NewUserController(service users.UserComponent) *UserController {
	return &UserController{service: service}
}

func (c *UserController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/user/:id", c.getByID)
}

func (c *UserController) getByID(ctx *gin.Context) {
	userID := ctx.Param("id")
	order, err := c.service.FindUserByID(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"order": order})
}
