package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"greenlight.dzenthai.net/internal/data"
	"greenlight.dzenthai.net/internal/mailer"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	cfg    config
	logger *slog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", getIntEnv("PORT", 4000), "Server port")
	flag.StringVar(&cfg.env, "env", getStringEnv("ENV", "development"), "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "dsn", getStringEnv("DSN", "localhost:5432/postgres"), "PostgreSQL data source name")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", getIntEnv("MAX_OPEN_CONNS", 25), "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", getIntEnv("MAX_IDLE_CONNS", 25), "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", getStringEnv("MAX_IDLE_TIME", "15m"), "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", getFloatEnv("LIMITER_RPS", 2), "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", getIntEnv("LIMITER_BURST", 4), "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", getBoolEnv("LIMITER_ENABLED", true), "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", getStringEnv("SMTP_HOST", ""), "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", getIntEnv("SMTP_PORT", 25), "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", getStringEnv("SMTP_USERNAME", ""), "SMTP Username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", getStringEnv("SMTP_PASSWORD", ""), "SMTP Password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", getStringEnv("SMTP_SENDER", ""), "SMTP Sender")
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(cfg)
	if err != nil {
		logger.Error("unable to establish a connection with the database", "err", err)
		os.Exit(1)
	}

	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)

	logger.Info("database connection pool established")

	m, err := mailer.NewMailer(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender)
	if err != nil {
		logger.Error("failed to create mailer", "err", err)
		os.Exit(1)
	}

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))

	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	app := &application{
		cfg:    cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: m,
	}

	if err = app.serve(); err != nil {
		logger.Error("server stopped unexpectedly", "err", err)
		os.Exit(1)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.db.dsn)

	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
