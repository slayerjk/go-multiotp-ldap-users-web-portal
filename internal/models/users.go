package models

import (
	"time"
)

// Define a new User struct. Notice how the field names and types align
// with the columns in the database "users" table?
type User struct {
	ID   int
	Name string
	// Email          string
	Login          string
	HashedPassword []byte
	Created        time.Time
}

// Define a new UserModel struct which wraps a database connection pool.
// type UserModel struct {
// 	DB *sql.DB
// }

// We'll use the Authenticate method to verify whether a user exists with
// the provided email address and password. This will return the relevant
// user ID if they do.
// func (m *UserModel) Authenticate(email, password string) (int, error) {

// func Authenticate(email, password string) (int, error) {
func Authenticate(login, password string) (int, error) {
	// Retrieve the id and hashed password associated with the given email. If
	// no matching email exists we return the ErrInvalidCredentials error.
	var id int
	// var hashedPassword []byte

	// stmt := "SELECT id, hashed_password FROM users WHERE email = ?"

	// // err := m.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword)
	// err := m.DB.QueryRow(stmt, login).Scan(&id, &hashedPassword)
	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {
	// 		return 0, ErrInvalidCredentials
	// 	} else {
	// 		return 0, err
	// 	}
	// }

	// Check whether the hashed password and plain-text password provided match.
	// If they don't, we return the ErrInvalidCredentials error.
	// err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	// if err != nil {
	// 	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
	// 		return 0, ErrInvalidCredentials
	// 	} else {
	// 		return 0, err
	// 	}
	// }

	// Otherwise, the password is correct. Return the user ID.
	return id, nil
}

// We'll use the Exists method to check if a user exists with a specific ID.
// func (m *UserModel) Exists(id int) (bool, error) {
func Exists(id int) (bool, error) {
	var exists bool

	// stmt := "SELECT EXISTS(SELECT true FROM users WHERE id = ?)"
	// err := m.DB.QueryRow(stmt, id).Scan(&exists)
	// return exists, err
	return exists, nil
}
