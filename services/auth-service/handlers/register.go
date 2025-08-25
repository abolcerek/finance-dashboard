package handlers

import (
	"auth-service/models"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"log"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)


type Request struct {
	Email string `json:"email"`
	Password string `json:"password"`
}


func NewHandler (db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	} 
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
		return
	}
	if len(req.Password) < 8{
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error when securing password"})
		return
	}
	req.Password = ""

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	u, err := models.InsertUser(ctx, db, req.Email, string(hashed))
	log.Printf("handler err: %T | %v", err, err)
	log.Printf("is duplicate? %v", errors.Is(err, models.ErrEmailExists))

	if errors.Is(err, models.ErrEmailExists) {
		c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		return
	}
	
	if errors.Is(err, context.DeadlineExceeded) {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "request timed out"})
		return
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}



	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "email": u.Email, "created_at": u.CreatedAt })
	}

	}
	