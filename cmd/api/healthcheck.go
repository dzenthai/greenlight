package main

import (
	"fmt"
	"net/http"
)

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "status: available\n")
	_, _ = fmt.Fprintf(w, "environment: %s\n", app.cfg.env)
	_, _ = fmt.Fprintf(w, "version: %s\n", version)
}
