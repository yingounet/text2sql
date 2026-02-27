package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"text2sql/internal/api"
	"text2sql/internal/config"
	"text2sql/internal/llmfactory"
	"text2sql/internal/text2sql"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.APIKey == "" {
		log.Fatal("api_key 未配置，请设置 config.api_key 或环境变量 API_KEY")
	}

	llmProvider, err := llmfactory.NewProviderFromConfig(&cfg.LLM)
	if err != nil {
		log.Fatalf("create llm provider: %v", err)
	}

	var store text2sql.ContextStore
	switch cfg.ContextStore {
	case "sqlite":
		sqliteStore, err := text2sql.NewSQLiteContextStore(cfg.Database.DSN)
		if err != nil {
			log.Fatalf("create sqlite context store: %v", err)
		}
		store = sqliteStore
	default:
		store = text2sql.NewMemoryContextStore()
	}

	validator := text2sql.NewSQLValidator()
	svc := text2sql.NewServiceWithContextStore(llmProvider, validator, 2, store)

	handler := api.NewHandler(svc, cfg.APIKey)

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

	// 优雅关闭
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint
		log.Println("Shutting down server...")

		// 关闭 ContextStore
		if err := store.Close(); err != nil {
			log.Printf("Error closing context store: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		} else {
			log.Println("Server gracefully stopped")
		}
	}()

	log.Printf("server listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
