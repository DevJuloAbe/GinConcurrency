package utils

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, status int, payload gin.H) {
	c.JSON(status, payload)
}

func Error(c *gin.Context, status int, message string, err error, extra gin.H) {
	payload := gin.H{
		"message": message,
	}

	if err != nil {
		payload["error"] = err.Error()
	}

	for key, value := range extra {
		payload[key] = value
	}

	c.JSON(status, payload)
}
