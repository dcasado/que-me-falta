package quemefalta

import "time"

type Product struct {
	ID   string
	Name string
}

type ProductService interface {
	CreateProduct(name string) error
	Products() ([]*Product, error)
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
