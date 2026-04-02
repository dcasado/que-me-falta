package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	quemefalta "github.com/dcasado/que-me-falta"
	"github.com/dcasado/que-me-falta/internal/html"
	"github.com/dcasado/que-me-falta/internal/static"
)

const cookieName string = "token"

type SessionHandler struct {
	SigningKey           string
	PasswordSHA256       string
	MaxSessionAgeSeconds int
	SessionService       quemefalta.SessionService
}

type ProductsHandler struct {
	ProductService quemefalta.ProductService
}

func Serve(listenAddress, listenPort string, sessionHandler SessionHandler, productsHandler ProductsHandler) *http.Server {
	serveMux := http.NewServeMux()

	resources := http.FileServer(http.FS(static.Resources()))
	serveMux.Handle("/resources/", http.StripPrefix("/resources/", resources))
	serveMux.Handle("/favicon.ico", resources)
	serveMux.HandleFunc("/", sessionHandler.Auth(productsHandler.Index))
	serveMux.HandleFunc("/product", sessionHandler.Auth(productsHandler.Product))
	serveMux.HandleFunc("/product/delete", sessionHandler.Auth(productsHandler.DeleteProduct))
	serveMux.HandleFunc("/login", sessionHandler.Login)
	serveMux.HandleFunc("/logout", sessionHandler.Auth(sessionHandler.Logout))

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", listenAddress, listenPort),
		Handler: serveMux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting the server: %s", err)
		}
	}()
	log.Printf("Started server listening on %s:%s", listenAddress, listenPort)

	return server
}

func (h *SessionHandler) Login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		err := html.Login(w)
		if err != nil {
			log.Printf("error serving login: %v", err)
		}
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		formPassword := r.FormValue("password")
		formPasswordSHA256 := sha256.Sum256([]byte(formPassword))
		encodedPasswordSHA256 := hex.EncodeToString(formPasswordSHA256[:])
		if strings.EqualFold(encodedPasswordSHA256, h.PasswordSHA256) {
			session, err := h.SessionService.CreateSession()
			if err != nil {
				log.Printf("unable to create session: %v", err)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			token := session.Token
			// Calculate a HMAC signature of the cookie name and value, using SHA256 and a secret key.
			mac := hmac.New(sha256.New, []byte(h.SigningKey))
			mac.Write([]byte(cookieName))
			mac.Write([]byte(token))
			signature := mac.Sum(nil)

			// Prepend the token with the HMAC signature and encode it to base64.
			cookieValue := base64.URLEncoding.EncodeToString([]byte(string(signature) + token))

			cookie := &http.Cookie{
				Name:     cookieName,
				Value:    cookieValue,
				MaxAge:   h.MaxSessionAgeSeconds,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
				HttpOnly: true,
			}
			http.SetCookie(w, cookie)

			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			err := html.Login(w)
			if err != nil {
				log.Printf("error serving login: %v", err)
			}
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (h *SessionHandler) Auth(authenticatedHandlerFunc http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		signedToken, err := base64.URLEncoding.DecodeString(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Check that the signed token is at least the size of the signature
		if len(signedToken) < sha256.Size {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Split apart the signature and the token.
		signature := signedToken[:sha256.Size]
		token := signedToken[sha256.Size:]

		// Recalculate the HMAC signature of the cookie name and the token.
		mac := hmac.New(sha256.New, []byte(h.SigningKey))
		mac.Write([]byte(cookieName))
		mac.Write([]byte(token))
		expectedSignature := mac.Sum(nil)

		if !hmac.Equal(signature, expectedSignature) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		session, err := h.SessionService.Session(string(token))
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if string(token) != session.Token || session.ExpirationTimestamp.Before(time.Now().UTC()) {
			err = h.SessionService.DeleteSession(string(token))
			if err != nil {
				log.Printf("could not delete session from DB: %v", err)
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		authenticatedHandlerFunc(w, r)
	})
}

func (h *SessionHandler) Logout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		signedToken, err := base64.URLEncoding.DecodeString(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Check that the signed token is at least the size of the signature
		if len(signedToken) < sha256.Size {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Split apart the signature and the token
		token := signedToken[sha256.Size:]
		err = h.SessionService.DeleteSession(string(token))
		if err != nil {
			log.Printf("could not delete session from DB: %v", err)
		}

		cookie = &http.Cookie{
			Name:   cookieName,
			MaxAge: -1,
		}
		http.SetCookie(w, cookie)

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (h *ProductsHandler) Index(w http.ResponseWriter, r *http.Request) {
	products, err := h.ProductService.Products()
	if err != nil {
		log.Printf("error retrieving products: %v", err)
	}
	err = html.Index(w, products)
	if err != nil {
		log.Printf("error serving index: %v", err)
	}
}

func (h *ProductsHandler) Product(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		formName := r.FormValue("name")

		err = h.ProductService.CreateProduct(formName)
		if err != nil {
			log.Printf("error creating a product: %v", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (h *ProductsHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		formID := r.FormValue("id")

		err = h.ProductService.DeleteProduct(formID)
		if err != nil {
			log.Printf("error deleting a product: %v", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}
