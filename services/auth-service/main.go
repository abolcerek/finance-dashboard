package main

import (
	// "github.com/gin-gonic/gin"
	// "net/http"
	"auth-service/handlers"
	"auth-service/handlers/middleware"
	"database/sql"
	"fmt"
	"log"
	"os"
	"github.com/joho/godotenv"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/gin-gonic/gin"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("No .env file found: %v", err)
	}
}

func main() {
	jwtSecret := os.Getenv("JWT_SECRET")
	db_config := os.Getenv("DB_CONN")
	if jwtSecret == ""{
		log.Fatal("JWT_SECRET is required")
	}
	if db_config == ""{
		log.Fatal("DB_CONN is required")
	}
	db, err := sql.Open("mssql", db_config)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	fmt.Println("Successfully connected and pinged the database")

	router := gin.Default()

	authMW := middleware.Auth(jwtSecret)
	router.POST("/register", handlers.NewHandler(db))
	router.POST("/login", handlers.AuthHandler(db, jwtSecret))
	router.GET("/me", authMW, handlers.MeHandler())
	ag := router.Group("/analytics")
	ag.Use(authMW)
	ag.GET("/summary", handlers.AnalyticsSummary(db))
	ag.GET("/cashflow", handlers.AnalyticsCashflow(db))
	ag.GET("/budget", handlers.AnalyticsBudgets(db))
	router.Run(":8080")

}
