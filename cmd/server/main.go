package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aadithya-md/split-expense/internal/config"
	"github.com/aadithya-md/split-expense/internal/router"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	r := router.NewRouter()

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.HttpServer.Address, cfg.HttpServer.Port),
		Handler:      r,
		ReadTimeout:  cfg.HttpServer.ReadTimeout,
		WriteTimeout: cfg.HttpServer.WriteTimeout,
		IdleTimeout:  cfg.HttpServer.IdleTimeout,
	}

	log.Printf("Starting server on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting server: %v", err)
	}
}
