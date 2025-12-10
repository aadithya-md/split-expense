package handler

import (
	"net/http"
)

// HealthCheckHandler returns a 200 OK for health checks.
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}
