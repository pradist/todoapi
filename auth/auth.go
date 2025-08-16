package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func AccessToken(signature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "todoapi",
			Subject:   "user_id",
		})

		ss, err := token.SignedString([]byte(signature))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": ss,
		})
	}
}
