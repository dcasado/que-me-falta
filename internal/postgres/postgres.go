package postgres

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
	_ "github.com/jackc/pgx/v5/stdlib"
)

// SessionService represents a PostgreSQL implementation of quemefalta.SessionService.
type SessionService struct {
	DB                   *sql.DB
	MaxSessionAgeSeconds int
}

type ProductService struct {
	DB *sql.DB
}

func Open(uri string) (*sql.DB, error) {
	db, err := sql.Open("pgx", uri)
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
		fmt.Println("error:", err)
		return nil, err
	}
	// Encode the byte array to an string with hexadecimal encoding
	token := hex.EncodeToString(b)
	expirationTimestamp := time.Now().UTC().Add(time.Duration(s.MaxSessionAgeSeconds) * time.Second)
	_, err = s.DB.Exec(`INSERT INTO sessions (token, expiration_timestamp) values ($1, $2)`, token, expirationTimestamp)
	if err != nil {
		return nil, fmt.Errorf("createSession: %v", err)
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
		return fmt.Errorf("deleteSession: %v", err)
	}
	return nil
}

// CreateProduct creates a new product.
func (p *ProductService) CreateProduct(name string, description string, quantity string) error {
	_, err := p.DB.Exec(`INSERT INTO products (name, description, quantity, added) values ($1, $2, $3, true)`, name, description, quantity)
	if err != nil {
		return fmt.Errorf("createProduct: %v", err)
	}
	return nil
}

// AddedProducts returns a list of products with the added field set to true.
func (p *ProductService) AddedProducts() ([]*quemefalta.Product, error) {
	var products []*quemefalta.Product
	rows, err := p.DB.Query(`SELECT id, name, description, quantity, added FROM products WHERE added ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("addedProducts: %v", err)
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var product quemefalta.Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.Quantity, &product.Added); err != nil {
			return nil, fmt.Errorf("addedProducts: %v", err)
		}
		products = append(products, &product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("addedProducts: %v", err)
	}
	return products, nil
}

// RemainingProducts retrieves a list of products with the added field set to false.
func (p *ProductService) RemainingProducts() ([]*quemefalta.Product, error) {
	var products []*quemefalta.Product
	rows, err := p.DB.Query(`SELECT id, name, description, quantity, added FROM products WHERE NOT added ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("remainingProducts: %v", err)
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var product quemefalta.Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.Quantity, &product.Added); err != nil {
			return nil, fmt.Errorf("remainingProducts: %v", err)
		}
		products = append(products, &product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("remainingProducts: %v", err)
	}
	return products, nil
}

// AddProduct modifies a product to set the added field to true
func (p *ProductService) AddProduct(id string) error {
	_, err := p.DB.Exec(`UPDATE products SET added = true WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("addProduct: %v", err)
	}
	return nil
}

// RemoveProduct modifies a product to set the added field to false
func (p *ProductService) RemoveProduct(id string) error {
	_, err := p.DB.Exec(`UPDATE products SET added = false WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("removeProduct: %v", err)
	}
	return nil
}

// Product returns a product for a given id
func (p *ProductService) Product(id string) (*quemefalta.Product, error) {
	var product quemefalta.Product
	row := p.DB.QueryRow(`SELECT id, name, description, quantity, added FROM products WHERE id = $1`, id)
	if err := row.Scan(&product.ID, &product.Name, &product.Description, &product.Quantity, &product.Added); err != nil {
		return nil, fmt.Errorf("product: %v", err)
	}
	return &product, nil
}

// EditProduct changes product properties
func (p *ProductService) EditProduct(id string, name string, description string, quantity string) error {
	_, err := p.DB.Exec(`UPDATE products SET name = $2, description = $3, quantity = $4 WHERE id = $1`, id, name, description, quantity)
	if err != nil {
		return fmt.Errorf("editProduct: %v", err)
	}
	return nil
}

// DeleteProduct deletes a product with a given id.
func (s *ProductService) DeleteProduct(id string) error {
	_, err := s.DB.Exec(`DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleteProduct: %v", err)
	}
	return nil
}
