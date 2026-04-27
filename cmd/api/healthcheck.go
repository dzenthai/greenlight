package main

import (
	"net/http"
)

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":      "available",
		"environment": app.cfg.env,
		"version":     version,
	}

	if err := app.writeJSON(w, http.StatusOK, data, nil); err != nil {
		http.Error(
			w,
			"the server encountered a problem and could not process your request",
			http.StatusInternalServerError,
		)
	}
}
