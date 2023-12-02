package html

import (
	"embed"
	"html/template"
	"io"

	quemefalta "github.com/dcasado/que-me-falta"
)

//go:embed *.html
var htmlFiles embed.FS

type indexParams struct {
	AddedProducts     []productParams
	RemainingProducts []productParams
}

type productParams struct {
	Id          string
	Name        string
	Description string
	Quantity    string
	Added       bool
}

func Login(w io.Writer) error {
	return parse("login.html").Execute(w, nil)
}

func Index(w io.Writer, addedProducts []*quemefalta.Product, remainingProducts []*quemefalta.Product) error {
	var indexParams indexParams
	for _, p := range addedProducts {
		productParam := productParams{
			Id:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Quantity:    p.Quantity,
			Added:       p.Added,
		}
		indexParams.AddedProducts = append(indexParams.AddedProducts, productParam)
	}
	for _, p := range remainingProducts {
		productParam := productParams{
			Id:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Quantity:    p.Quantity,
			Added:       p.Added,
		}
		indexParams.RemainingProducts = append(indexParams.RemainingProducts, productParam)
	}
	return parse("index.html").Execute(w, indexParams)
}

func Product(w io.Writer, p *quemefalta.Product) error {
	productParam := productParams{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Quantity:    p.Quantity,
		Added:       p.Added,
	}
	return parse("product.html").Execute(w, productParam)
}

func parse(file string) *template.Template {
	return template.Must(
		template.New("layout.html").ParseFS(htmlFiles, "layout.html", file))
}
