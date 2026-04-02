package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	quemefalta "github.com/dcasado/que-me-falta"
	_ "github.com/mattn/go-sqlite3"
)

// SessionService represents a SQL implementation of quemefalta.SessionService.
type SessionService struct {
	DB                   *sql.DB
	MaxSessionAgeSeconds int
}

type ProductService struct {
	DB *sql.DB
}

func Open(uri string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", uri)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	return db, nil
}

// CreateSession creates a new session.
func (s *SessionService) CreateSession() (*quemefalta.Session, error) {
	// Make a byte array of size 32
	b := make([]byte, 32)
	// Populate the array with random numbers
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("error reading random bytes:", err)
		return nil, err
	}
	// Encode the byte array to an string with hexadecimal encoding
	token := hex.EncodeToString(b)
	expirationTimestamp := time.Now().UTC().Add(time.Duration(s.MaxSessionAgeSeconds) * time.Second)
	_, err = s.DB.Exec(`INSERT INTO sessions (token, expiration_timestamp) values ($1, $2)`, token, expirationTimestamp)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %v", err)
	}
	return &quemefalta.Session{Token: token, ExpirationTimestamp: expirationTimestamp}, nil
}

// Session returns a session for a given token.
func (s *SessionService) Session(token string) (*quemefalta.Session, error) {
	var session quemefalta.Session
	row := s.DB.QueryRow(`SELECT token, expiration_timestamp FROM sessions WHERE token = $1`, token)
	if err := row.Scan(&session.Token, &session.ExpirationTimestamp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		log.Fatalf("could not retrieve session: %v", err)
	}
	return &session, nil
}

// DeleteSession deletes a session.
func (s *SessionService) DeleteSession(token string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE token = $1`, token)
	if err != nil {
		return fmt.Errorf("error deleting session: %v", err)
	}
	return nil
}

// CreateProduct creates a new product.
func (p *ProductService) CreateProduct(name string) error {
	_, err := p.DB.Exec(`INSERT INTO products (name) values ($1)`, name)
	if err != nil {
		return fmt.Errorf("error inserting product with name \"%s\" into products: %v", name, err)
	}
	return nil
}

// Products returns a list of products
func (p *ProductService) Products() ([]*quemefalta.Product, error) {
	var products []*quemefalta.Product
	rows, err := p.DB.Query(`SELECT id, name FROM products ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("error querying products: %v", err)
	}
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var product quemefalta.Product
		if err := rows.Scan(&product.ID, &product.Name); err != nil {
			return nil, fmt.Errorf("error scanning product row: %v", err)
		}
		products = append(products, &product)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("error closing product rows: %v", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during product rows iteration: %v", err)
	}
	return products, nil
}

// DeleteProduct deletes a product with a given id.
func (p *ProductService) DeleteProduct(id string) error {
	_, err := p.DB.Exec(`DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("error deleting product with id \"%s\": %v", id, err)
	}
	return nil
}
