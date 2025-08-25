package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func MeHandler () gin.HandlerFunc {
	return func( c*gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized access"})
			return
		}
		email, exists := c.Get("email")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized access"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id": userID,
			"email": email,
		})
				
	}
}