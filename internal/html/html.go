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
	Products []productParams
}

type productParams struct {
	ID   string
	Name string
}

func Login(w io.Writer) error {
	return parse("login.html").Execute(w, nil)
}

func Index(w io.Writer, products []*quemefalta.Product) error {
	var indexParams indexParams
	for _, p := range products {
		productParam := productParams{
			ID:   p.ID,
			Name: p.Name,
		}
		indexParams.Products = append(indexParams.Products, productParam)
	}
	return parse("index.html").Execute(w, indexParams)
}

func parse(file string) *template.Template {
	return template.Must(
		template.New("layout.html").ParseFS(htmlFiles, "layout.html", file))
}
