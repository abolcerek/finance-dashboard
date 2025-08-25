package handlers

import (
	"auth-service/models"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
)

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func AuthHandler(db *sql.DB, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var l Login
		if err := c.ShouldBindJSON(&l); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
		l.Email = strings.ToLower(strings.TrimSpace(l.Email))
		l.Password = strings.TrimSpace(l.Password)

		if l.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
			return
		}

		if l.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		u, hashedPassword, err := models.GetUserByEmail(ctx, db, l.Email)

		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
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

		Err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(l.Password))

		if Err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		now := time.Now().Unix()
		exp := time.Now().Add(1*time.Hour).Unix()

		claims := jwt.MapClaims {
			"sub": u.ID,
			"email": u.Email,
			"iat": now,
			"exp": exp, 
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(jwtSecret))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.SetCookie(
			"auth_token",
			signed,
			3600,
			"/",
			"",
			//  secure
			false, 
			// httpOnly
			true,
		)

		c.JSON(http.StatusOK, gin.H{"id": u.ID, "email": u.Email})
		}
	}

