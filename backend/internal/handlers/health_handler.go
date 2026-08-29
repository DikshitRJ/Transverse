package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler returns an HTTP handler that checks database connectivity and pool health metrics.
func HealthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		stat := pool.Stat()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"pool_total":    stat.TotalConns(),
			"pool_idle":     stat.IdleConns(),
			"pool_acquired": stat.AcquiredConns(),
		})
	}
}
