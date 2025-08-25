package models

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"github.com/denisenkom/go-mssqldb"
)

type User struct {
	ID             int64     `json:"id"`
	Email          string    `json:"email"` 
	CreatedAt      time.Time `json:"created_at"`
}


var ErrEmailExists = errors.New("email already exists")
func InsertUser (ctx context.Context, db *sql.DB, email string, hashedPassword string) (User, error) {
	sqlStatement := `
	INSERT INTO users (email, hashed_password)
	OUTPUT INSERTED.id, INSERTED.email, INSERTED.created_at
	VALUES (?, ?)`


	var u User
	row :=db.QueryRowContext(ctx, sqlStatement, email, hashedPassword)
	err := row.Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err == nil {
		return u, nil
	}
	var mePtr *mssql.Error
	if errors.As(err, &mePtr) && (mePtr.Number == 2627 || mePtr.Number == 2601) {
		return User{}, ErrEmailExists
	}
	var meVal mssql.Error
	if errors.As(err, &meVal) && (meVal.Number == 2627 || meVal.Number == 2601) {
		return User{}, ErrEmailExists
	}

	return User{}, err
}


func GetUserByEmail(ctx context.Context, db *sql.DB, email string) (User, string, error){
	sqlStatement := `
	SELECT id, email, hashed_password, created_at FROM users WHERE email = ?`

	var u User
	var hash string
	row :=db.QueryRowContext(ctx, sqlStatement, email)
	err := row.Scan(&u.ID, &u.Email, &hash, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, "", err
		}
		return User{}, "", err
	}	

	return u, hash, nil





}
	