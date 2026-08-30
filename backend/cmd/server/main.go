// Package main provides the application entrypoint, dependency wiring, router setup, and graceful shutdown.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"transverse/internal/cache"
	"transverse/internal/config"
	"transverse/internal/connectors"
	"transverse/internal/database"
	"transverse/internal/evidence"
	"transverse/internal/graph"
	"transverse/internal/handlers"
	"transverse/internal/jobs"
	"transverse/internal/llm"
	"transverse/internal/middleware"
	"transverse/internal/objectstore"
	"transverse/internal/realtime"
	"transverse/internal/repository"
	"transverse/internal/roadmap"
	"transverse/internal/scraper"
	"transverse/internal/services"
	"transverse/internal/services/ingest"
)

// evidenceJobQueueAdapter adapts the generic Redis-backed jobs.Queue to the narrower
// evidence.JobQueue interface that internal/evidence/service.go depends on (it only
// needs to enqueue one job type and get an ID back).
type evidenceJobQueueAdapter struct {
	q jobs.Queue
}

func (a *evidenceJobQueueAdapter) EnqueueHypothesisGeneration(ctx context.Context, userID string) (string, error) {
	job := &jobs.Job{
		ID:      uuid.New().String(),
		UserID:  userID,
		JobType: "hypothesis_generation",
	}
	if err := a.q.Enqueue(ctx, job); err != nil {
		return "", err
	}
	return job.ID, nil
}

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
	pool, err := database.NewPostgresPool(ctx, cfg)
	cancel()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("connected to postgres database pool")

	// 3.5 Setup Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   cfg.RedisDB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Warn("failed to ping redis", "error", err)
	} else {
		slog.Info("connected to redis", "addr", cfg.RedisAddr)
	}

	// 4. Setup cache
	var appCache cache.Cache
	if cfg.CacheEnabled {
		if rdb != nil {
			appCache = cache.NewRedisCache(rdb)
			slog.Info("redis cache enabled")
		} else {
			appCache = cache.NewMemoryCache()
			slog.Info("in-memory cache enabled (redis unavailable)")
		}
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
		pool, problemRepo, statsRepo, sessionRepo, userRepo, probStats,
		graphSvc, appCache, cfg, rdb,
	)
	ingestRepo := repository.NewIngestRepo(pool)
	ingestSvc := ingest.NewService(ingestRepo, graphSvc)
	problemScraper := scraper.NewUnifiedScraper(15 * time.Second)
	roadmapRepo := repository.NewRoadmapRepo(pool)
	llmClient := llm.NewZaiClient(cfg, rdb)
	roadmapSvc, _ := roadmap.NewService(roadmapRepo, userRepo, problemRepo, statsRepo, topicGraph, llmClient, rdb, "")

	// 8. Build handlers
	practiceH := handlers.NewPracticeHandler(practiceSvc, judge0Svc, problemRepo, problemScraper)
	userH := handlers.NewUserHandler(userRepo, statsRepo, sessionRepo)
	ingestH := handlers.NewIngestHandler(ingestSvc)
	oauthRepo := repository.NewOAuthRepo(pool)
	authH := handlers.NewAuthHandler(cfg, oauthRepo, userRepo, appCache)
	roadmapH := handlers.NewRoadmapHandler(roadmapSvc)

	// Jobs & Realtime
	jobQueue := jobs.NewQueue(rdb)
	workerPool := jobs.NewWorkerPool(jobQueue, rdb)
	workerPool.Start(context.Background()) // Note: context should be proper in real prod

	jobsH := jobs.NewHandler(jobQueue)
	realtimeH := realtime.NewHandler(rdb)

	// Evidence pipeline: MinIO-backed object store + Postgres repo + connectors.
	// The server must still boot even if MinIO isn't reachable yet (mirrors the Redis
	// ping-and-warn pattern above) — evidence upload endpoints will simply error until
	// it comes up.
	minioClient := objectstore.NewMinIOClient(cfg)
	{
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := minioClient.Ping(pingCtx); err != nil {
			slog.Warn("minio unreachable at startup; evidence upload endpoints will fail until it recovers", "endpoint", cfg.MinIOEndpoint, "error", err)
		} else {
			slog.Info("connected to minio", "endpoint", cfg.MinIOEndpoint)
			if err := minioClient.EnsureBucket(pingCtx); err != nil {
				slog.Warn("failed to ensure minio bucket exists", "bucket", cfg.MinIOBucket, "error", err)
			}
		}
		pingCancel()
	}

	evidenceRepo := repository.NewEvidenceRepo(pool)
	githubConnector := connectors.NewGithubConnector(cfg)
	leetcodeConnector := connectors.NewLeetcodeConnector(cfg)
	codeforcesConnector := connectors.NewCodeforcesConnector(cfg)
	evidenceQueue := &evidenceJobQueueAdapter{q: jobQueue}
	evidenceSvc := evidence.NewService(evidenceRepo, minioClient, evidenceQueue, githubConnector, leetcodeConnector, codeforcesConnector)
	evidenceH := handlers.NewEvidenceHandler(evidenceSvc)

	// 9. Setup router
	r := chi.NewRouter()
	r.Use(middleware.CORS(cfg.FrontendOrigin))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.CleanPath)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(middleware.RateLimit(rdb, 120, time.Minute))

	// Health check endpoint (unauthenticated)
	r.Get("/health", handlers.HealthHandler(pool))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authH.Register)
			r.Post("/signup", authH.Register)
			r.Post("/login", authH.Login)
			r.Post("/signin", authH.Login)
			r.Get("/oauth/{provider}/redirect", authH.Redirect)
			r.Get("/oauth/{provider}/callback", authH.Callback)
			r.Post("/refresh", authH.Refresh)
			r.Post("/logout", authH.Logout)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret, appCache))
			r.Get("/auth/me", authH.Me)

			// Practice endpoints
			r.Route("/practice", func(r chi.Router) {
				r.Post("/start", practiceH.StartSession)
				r.Post("/submit", practiceH.SubmitAnswer)
				r.Post("/skip", practiceH.SkipProblem)
				r.Post("/close", practiceH.CloseSession)
				r.Get("/session/{id}", practiceH.GetSession)
				r.Post("/{id}/hint", practiceH.RequestHint)
				r.Get("/{id}/error-analysis", practiceH.GetErrorAnalysis)
				r.Get("/similar", practiceH.GetSimilar)
				r.Get("/topics", practiceH.GetTopics)
			})

			// Evidence intake endpoints (resume/codebase upload + connector pull)
			r.Route("/evidence", func(r chi.Router) {
				r.Post("/upload-url", evidenceH.HandleUploadURL)
				r.Post("/{id}/confirm", evidenceH.HandleConfirmUpload)
				r.Post("/github", evidenceH.HandleGithub)
				r.Post("/leetcode", evidenceH.HandleLeetcode)
				r.Post("/codeforces", evidenceH.HandleCodeforces)
			})

		// Dynamic Progressive Roadmap endpoints
		r.Route("/roadmap", func(r chi.Router) {
			r.Get("/", roadmapH.GetCurrentRoadmap)
			r.Get("/me", roadmapH.GetCurrentRoadmap)
			r.Post("/nodes/{id}/complete", roadmapH.CompleteNode)
			r.Post("/nodes/{id}/test-out", roadmapH.TestOutNode)
			r.Post("/generate", roadmapH.GenerateRoadmap)
		})

		// Code execution endpoints
		r.Route("/execute", func(r chi.Router) {
			r.Post("/", practiceH.Execute)
			r.Post("/batch", practiceH.ExecuteBatch)
			r.Get("/{token}", practiceH.GetVerdict)
		})

		// Problem repository search and scraping
		r.Get("/problems/search", practiceH.SearchProblems)
		r.Post("/problems/scrape", practiceH.ScrapeProblem)

		// Realtime SSE endpoint
		r.Get("/events/stream", realtimeH.StreamEvents)
		
		// Jobs polling endpoint
		r.Get("/jobs/{id}", jobsH.GetJob)

		// User profile & history endpoints
		r.Route("/user", func(r chi.Router) {
			r.Get("/profile", userH.GetProfile)
			r.Get("/history", userH.GetHistory)
		})

		// Admin endpoints
		r.Route("/admin", func(r chi.Router) {
			r.Post("/tutorials/ingest", ingestH.IngestTutorials)
			r.Post("/roadmaps/ingest", ingestH.IngestRoadmaps)
		})
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
