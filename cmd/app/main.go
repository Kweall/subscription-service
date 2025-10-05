package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subscription-service/internal/api"
	"subscription-service/internal/config"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Info().Msg("Starting subscriptions service")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)

	var db *sql.DB
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		wait := time.Duration(2*i+1) * time.Second
		log.Warn().Err(err).Msgf("DB connect failed, retrying in %s", wait)
		time.Sleep(wait)
	}
	if err != nil {
		log.Fatal().Err(err).Msg("Could not connect to DB")
	}
	defer db.Close()

	repo := repository.NewPGRepo(db)
	svc := service.NewSubscriptionService(repo)
	handler := api.NewHandler(svc)

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/docs/openapi.yaml", handler.OpenAPIDoc)

	r.Route("/subscriptions", func(r chi.Router) {
		r.Post("/", handler.CreateSubscription)
		r.Get("/", handler.ListSubscriptions)
		r.Get("/total", handler.GetTotalCost)
		r.Get("/{id}", handler.GetSubscriptionByID)
		r.Put("/{id}", handler.UpdateSubscription)
		r.Delete("/{id}", handler.DeleteSubscription)
	})

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: r,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Msgf("HTTP server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	<-stop
	log.Info().Msg("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server Shutdown Failed")
	}
	log.Info().Msg("Server exited properly")
}
