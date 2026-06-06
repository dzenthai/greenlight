package main

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/realclientip/realclientip-go"
	"golang.org/x/time/rate"
	"greenlight.dzenthai.net/internal/data"
	"greenlight.dzenthai.net/internal/validator"
)

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		origin := r.Header.Get("Origin")

		if origin != "" {
			for _, v := range app.cfg.cors.trustedOrigins {
				if origin == v {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Allow-Methods") != "" {
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						w.WriteHeader(http.StatusOK)
					}
					break
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimiting(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)
	strategy, err := realclientip.NewRightmostNonPrivateStrategy("X-Forwarded-For")
	if err != nil {
		panic(err)
	}
	if app.cfg.limiter.enabled {
		go func() {
			for {
				time.Sleep(time.Minute)

				mu.Lock()

				for ip, cl := range clients {
					if time.Since(cl.lastSeen) > 3*time.Minute {
						delete(clients, ip)
					}
				}

				mu.Unlock()
			}
		}()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.cfg.limiter.enabled {
			ip := strategy.ClientIP(r.Header, r.RemoteAddr)

			mu.Lock()
			if _, found := clients[ip]; !found {
				rps := app.cfg.limiter.rps
				burst := app.cfg.limiter.burst
				clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
			}

			clients[ip].lastSeen = time.Now()

			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}

			mu.Unlock()

		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]

		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		user, err := app.models.UserModel.GetForTokens(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		r = app.contextSetUser(r, user)

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		permission, err := app.models.PermissionModel.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		if !permission.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	return app.requireActivatedUser(fn)
}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func (mw *metricsResponseWriter) WriteHeader(statusCode int) {
	mw.ResponseWriter.WriteHeader(statusCode)
	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

func (mw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !mw.headerWritten {
		mw.statusCode = 200
		mw.headerWritten = true
	}

	return mw.ResponseWriter.Write(b)
}

func (mw *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return mw.ResponseWriter
}

func (app *application) metrics(next http.Handler) http.Handler {
	var (
		totalRequestsReceived           = expvar.NewInt("total_requests_received")
		totalResponsesSent              = expvar.NewInt("total_responses_sent")
		totalProcessingTimeMicroseconds = expvar.NewInt("total_processing_time_μs")
		totalResponsesSentByStatus      = expvar.NewMap("total_responses_sent_by_status")
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		totalRequestsReceived.Add(1)

		mw := &metricsResponseWriter{ResponseWriter: w}

		next.ServeHTTP(mw, r)

		totalResponsesSent.Add(1)

		totalResponsesSentByStatus.Add(strconv.Itoa(mw.statusCode), 1)

		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	})
}
