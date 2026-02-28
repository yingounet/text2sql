package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"text2sql/internal/api"
	"text2sql/internal/config"
	"text2sql/internal/llm"
	"text2sql/internal/llmfactory"
	"text2sql/internal/logger"
	"text2sql/internal/text2sql"
)

func main() {
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger.SetLogger(slog.New(logHandler))

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	if len(cfg.APIKeys) == 0 {
		logger.Error("api_key 未配置")
		os.Exit(1)
	}

	llmProvider, err := llmfactory.NewProviderFromConfig(&cfg.LLM)
	if err != nil {
		logger.Error("create llm provider failed", "error", err)
		os.Exit(1)
	}

	cachedProvider := llm.NewCachedProvider(llmProvider, 5*time.Minute)

	var store text2sql.ContextStore
	switch cfg.ContextStore {
	case "sqlite":
		sqliteStore, err := text2sql.NewSQLiteContextStore(cfg.Database.DSN)
		if err != nil {
			logger.Error("create sqlite context store failed", "error", err)
			os.Exit(1)
		}
		store = sqliteStore
	default:
		store = text2sql.NewMemoryContextStore()
	}

	validator := text2sql.NewSQLValidator()
	svc := text2sql.NewServiceWithContextStore(cachedProvider, validator, 2, store)

	handler := api.NewHandler(svc, cfg.APIKeys)

	r := chi.NewRouter()
	handler.Routes(r)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint
		logger.Info("Shutting down server...")

		if err := store.Close(); err != nil {
			logger.Error("Error closing context store", "error", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("Server shutdown error", "error", err)
		} else {
			logger.Info("Server gracefully stopped")
		}
	}()

	logger.Info("server listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
