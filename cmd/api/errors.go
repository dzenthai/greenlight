package main

import (
	"fmt"
	"net/http"
	"time"
)

func (app *application) logError(r *http.Request, err error) {
	app.logger.Error("error occurs", "err", err)
}

func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	e := struct {
		StatusCode int       `json:"status_Code"`
		Error      any       `json:"error"`
		Timestamp  time.Time `json:"timestamp"`
	}{
		StatusCode: status,
		Error:      message,
		Timestamp:  time.Now(),
	}

	err := app.writeJSON(w, status, e, nil)
	if err != nil {
		app.logError(r, err)
		return
	}
}

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}

func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}
