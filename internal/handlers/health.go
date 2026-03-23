package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/version"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "ok",
		Version:   version.Version,
		Commit:    version.Commit,
		BuildDate: version.BuildDate,
	})
}
