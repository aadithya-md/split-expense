package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aadithya-md/split-expense/internal/config"
	"github.com/aadithya-md/split-expense/internal/repository"
	"github.com/aadithya-md/split-expense/internal/router"
	"github.com/aadithya-md/split-expense/internal/service"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	db, err := sql.Open("mysql", cfg.SQLDb.ConnectionString)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}
	defer db.Close()

	// Ping the database to verify the connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	log.Println("Successfully connected to the database!")

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)

	balanceRepo := repository.NewBalanceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db, balanceRepo)
	expenseService := service.NewExpenseService(expenseRepo, userService, balanceRepo)

	r := router.NewRouter(userService, expenseService)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.HttpServer.Address, cfg.HttpServer.Port),
		Handler:      r,
		ReadTimeout:  cfg.HttpServer.ReadTimeout,
		WriteTimeout: cfg.HttpServer.WriteTimeout,
		IdleTimeout:  cfg.HttpServer.IdleTimeout,
	}

	// Create a channel to listen for OS signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()
	log.Printf("Starting server on %s", srv.Addr)

	<-done // Block until an OS signal is received
	log.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped.")
}
