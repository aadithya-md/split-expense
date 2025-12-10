package router

import (
	"github.com/aadithya-md/split-expense/internal/handler"
	"github.com/aadithya-md/split-expense/internal/service"
	"github.com/gorilla/mux"
)

func NewRouter(userService service.UserService) *mux.Router {
	r := mux.NewRouter()

	healthHandler := handler.HealthCheckHandler
	userHandler := handler.NewUserHandler(userService)

	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/users", userHandler.CreateUserHandler).Methods("POST")
	r.HandleFunc("/users/{id}", userHandler.GetUserHandler).Methods("GET")
	r.HandleFunc("/users/by-email", userHandler.GetUserByEmailHandler).Methods("GET")

	return r
}
