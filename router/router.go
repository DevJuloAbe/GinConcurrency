package router

import (
	"gingonic-concurrency/controller"
	"gingonic-concurrency/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(userController *controller.UserController) *gin.Engine {
	r := gin.New()

	r.Use(middleware.Logger(), gin.Recovery())

	r.POST("/seed/users", userController.SeedUsers)
	r.GET("/fetch/users", userController.FetchUsers)
	r.GET("/fetch/users/channel", userController.FetchUsersByChannel)

	return r
}
