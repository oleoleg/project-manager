package handlers

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct {
	AppEnv string
}

func NewHealthHandler(appEnv string) *HealthHandler {
	return &HealthHandler{AppEnv: appEnv}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"env":    h.AppEnv,
	})
}
