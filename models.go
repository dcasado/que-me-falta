package quemefalta

import "time"

type Product struct {
	ID          string
	Name        string
	Description string
	Quantity    string
	Added       bool
}

type ProductService interface {
	CreateProduct(name string, description string, quantity string) error
	AddedProducts() ([]*Product, error)
	RemainingProducts() ([]*Product, error)
	AddProduct(id string) error
	RemoveProduct(id string) error
	Product(id string) (*Product, error)
	EditProduct(id string, name string, description string, quantity string) error
	DeleteProduct(id string) error
}

type Session struct {
	Token               string
	ExpirationTimestamp time.Time
}

type SessionService interface {
	CreateSession() (*Session, error)
	Session(token string) (*Session, error)
	DeleteSession(token string) error
}
