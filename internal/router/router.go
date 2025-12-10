package router

import (
	"github.com/aadithya-md/split-expense/internal/handler"
	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/health", handler.HealthCheckHandler).Methods("GET")

	return r
}
