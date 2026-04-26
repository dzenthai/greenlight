package main

import (
	"encoding/json"
	"net/http"
)

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":      "available",
		"environment": app.cfg.env,
		"version":     version,
	}
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(data)
}
