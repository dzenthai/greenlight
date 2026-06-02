package main

import (
	"expvar"
	"net/http"
)

func (app *application) routes() http.Handler {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthcheck", app.healthcheckHandler)

	mux.HandleFunc("GET /v1/movies", app.requirePermission("movie:read", app.listMoviesHandler))
	mux.HandleFunc("POST /v1/movies", app.requirePermission("movie:write", app.createMovieHandler))
	mux.HandleFunc("GET /v1/movies/{id}", app.requirePermission("movie:read", app.showMovieHandler))
	mux.HandleFunc("PATCH /v1/movies/{id}", app.requirePermission("movie:write", app.updateMovieHandler))
	mux.HandleFunc("DELETE /v1/movies/{id}", app.requirePermission("movie:write", app.deleteMovieHandler))

	mux.HandleFunc("POST /v1/users", app.registerUserHandler)
	mux.HandleFunc("PUT /v1/users/activated", app.activateUserHandler)
	mux.HandleFunc("POST /v1/tokens/authentication", app.createAuthenticationTokenHandler)

	mux.Handle("GET /debug/vars", expvar.Handler())

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimiting(app.authenticate(mux)))))
}
