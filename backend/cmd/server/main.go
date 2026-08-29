// Package main provides the application entrypoint, dependency wiring, router setup, and graceful shutdown.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"transverse/internal/cache"
	"transverse/internal/config"
	"transverse/internal/database"
	"transverse/internal/graph"
	"transverse/internal/handlers"
	"transverse/internal/middleware"
	"transverse/internal/repository"
	"transverse/internal/services"
)

func main() {
	// 1. Load config
	cfg := config.Load()

	// 2. Setup structured logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("starting transverse adaptive heuristic engine backend...")

	// 3. Connect to database pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := database.NewPool(ctx, cfg)
	cancel()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("connected to postgres database pool")

	// 4. Setup cache
	var appCache cache.Cache
	if cfg.CacheEnabled {
		appCache = cache.NewMemoryCache()
		slog.Info("in-memory cache enabled")
	} else {
		appCache = cache.NewNoopCache()
		slog.Info("cache disabled (using noop cache)")
	}

	// 5. Load topic graph
	topicGraph, err := graph.NewTopicGraph(cfg.TopicsGraphPath)
	if err != nil {
		slog.Error("failed to load topics graph", "path", cfg.TopicsGraphPath, "error", err)
		os.Exit(1)
	}
	slog.Info("loaded topic knowledge graph", "topics_count", len(topicGraph.GetAllTopics()))

	// 6. Build repos
	problemRepo := repository.NewProblemRepo(pool, appCache)
	userRepo := repository.NewUserRepo(pool, appCache)
	sessionRepo := repository.NewSessionRepo(pool)
	statsRepo := repository.NewStatsRepo(pool, appCache)
	probStats := repository.NewProblemStatsRepo(pool, appCache)

	// 7. Build services
	graphSvc := services.NewGraphService(topicGraph)
	judge0Svc := services.NewJudge0Service(cfg)
	practiceSvc := services.NewPracticeService(
		problemRepo, statsRepo, sessionRepo, userRepo, probStats,
		graphSvc, appCache, pool, judge0Svc,
	)

	// 8. Build handlers
	practiceH := handlers.NewPracticeHandler(practiceSvc, judge0Svc)
	userH := handlers.NewUserHandler(userRepo, statsRepo, sessionRepo)

	// 9. Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.CleanPath)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(middleware.RateLimit(120, time.Minute))

	// Health check endpoint (unauthenticated)
	r.Get("/health", handlers.HealthHandler(pool))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret))

		// Practice endpoints
		r.Route("/practice", func(r chi.Router) {
			r.Post("/start", practiceH.StartSession)
			r.Post("/submit", practiceH.SubmitAnswer)
			r.Post("/skip", practiceH.SkipProblem)
			r.Post("/close", practiceH.CloseSession)
			r.Get("/session/{id}", practiceH.GetSession)
			r.Get("/similar", practiceH.GetSimilar)
			r.Get("/topics", practiceH.GetTopics)
		})

		// Code execution endpoints
		r.Route("/execute", func(r chi.Router) {
			r.Post("/", practiceH.Execute)
			r.Get("/{token}", practiceH.GetVerdict)
		})

		// Problem repository search
		r.Get("/problems/search", practiceH.SearchProblems)

		// User profile & history endpoints
		r.Route("/user", func(r chi.Router) {
			r.Get("/profile", userH.GetProfile)
			r.Get("/history", userH.GetHistory)
		})
	})

	// Format listen address
	addr := cfg.Port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	// 10. Start server with graceful shutdown
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server fatal error", "error", err)
			os.Exit(1)
		}
	}()

	// Stale session cleanup goroutine (every 30 min)
	cleanupTicker := time.NewTicker(30 * time.Minute)
	defer cleanupTicker.Stop()
	go func() {
		for range cleanupTicker.C {
			n, err := sessionRepo.CleanupStale(context.Background(), 4*time.Hour)
			if err != nil {
				slog.Warn("stale session cleanup failed", "error", err)
			} else if n > 0 {
				slog.Info("stale sessions abandoned", "count", n)
			}
		}
	}()

	// Graceful shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("server stopped")
}
