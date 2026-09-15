
package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	auth "github.com/JerryJeager/raglearn/internal/http"
)

func JwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		id, err := auth.ValidateToken(c)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":     "Bad request",
				"message":    "Authentication failed",
				"statusCode": http.StatusUnauthorized,
			})
			fmt.Println(err)
			c.Abort()
			return
		}
		c.Set("user_id", id)

		c.Next()
	}
}

	