package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/dcasado/que-me-falta/internal/database"
	"github.com/dcasado/que-me-falta/internal/http"
)

const (
	databaseURIVariable          string = "DATABASE_URI"
	signingKeyVariable           string = "SIGNING_KEY"
	passwordSHA256Variable       string = "PASSWORD_SHA256"
	maxSessionAgeSecondsVariable string = "MAX_SESSION_AGE_SECONDS"
	listenAddressVariable        string = "LISTEN_ADDRESS"
	listenPortVariable           string = "LISTEN_PORT"
)

func main() {
	databaseURI, present := os.LookupEnv(databaseURIVariable)
	if !present {
		log.Fatalf("%s is required", databaseURIVariable)
	}

	// Connect to database.
	db, err := database.Open(databaseURI)
	if err != nil {
		log.Fatalf("error connecting to database: %v", err)
	}
	err = database.Migrate(db)
	if err != nil {
		log.Fatalf("error during migration %v", err)
	}

	maxSessionAgeSecondsStr, present := os.LookupEnv(maxSessionAgeSecondsVariable)
	if !present {
		maxSessionAgeSecondsStr = "300" // 5 minutes
	}
	maxSessionAgeSeconds, err := strconv.Atoi(maxSessionAgeSecondsStr)
	if err != nil {
		log.Fatalf("could not parse %s: %v", maxSessionAgeSecondsVariable, err)
	}
	ss := &database.SessionService{
		DB:                   db,
		MaxSessionAgeSeconds: maxSessionAgeSeconds,
	}

	signingKey, present := os.LookupEnv(signingKeyVariable)
	if !present {
		log.Fatalf("%s is required", signingKeyVariable)
	}
	passwordSHA256, present := os.LookupEnv(passwordSHA256Variable)
	if !present {
		log.Fatalf("%s is required", passwordSHA256Variable)
	}
	sh := http.SessionHandler{
		SigningKey:           signingKey,
		PasswordSHA256:       passwordSHA256,
		MaxSessionAgeSeconds: maxSessionAgeSeconds,
		SessionService:       ss,
	}

	ps := &database.ProductService{
		DB: db,
	}

	ph := http.ProductsHandler{
		ProductService: ps,
	}

	listenAddress, present := os.LookupEnv(listenAddressVariable)
	if !present {
		listenAddress = "127.0.0.1"
	}
	listenPort, present := os.LookupEnv(listenPortVariable)
	if !present {
		listenPort = "8080"
	}

	// Attach HTTP handlers to HTTP server
	server := http.Serve(listenAddress, listenPort, sh, ph)

	// Handle gracefull shutdown
	errC := make(chan error, 1)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-ctx.Done()

		log.Println("Shutdown signal received")

		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer func() {
			stop()
			cancel()
			close(errC)
			db.Close()
		}()

		server.SetKeepAlivesEnabled(false)

		if err := server.Shutdown(ctxTimeout); err != nil {
			errC <- err
		}

		log.Println("Shutdown completed")
	}()

	if err := <-errC; err != nil {
		log.Fatalln("error", err)
	}
	log.Print("Exited properly")
}
