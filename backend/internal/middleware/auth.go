// Package middleware provides HTTP middlewares for authentication, rate limiting, and request processing.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "userID"
	// UsernameKey is the context key for the authenticated user's username.
	UsernameKey contextKey = "username"
)

// Auth validates Bearer JWT tokens and injects userID and username into the request context.
// If BYPASS_AUTH is set to "true" or "1", or in development mode, it injects a default dev user ("dev-user-001").
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bypass := os.Getenv("BYPASS_AUTH") == "true" || os.Getenv("BYPASS_AUTH") == "1"

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				if bypass || jwtSecret == "" || jwtSecret == "change-me-in-production" {
					ctx := context.WithValue(r.Context(), UserIDKey, "dev-user-001")
					ctx = context.WithValue(ctx, UsernameKey, "dev-user")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"missing or invalid authorization header"}`))
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				if bypass {
					ctx := context.WithValue(r.Context(), UserIDKey, "dev-user-001")
					ctx = context.WithValue(ctx, UsernameKey, "dev-user")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid or expired token"}`))
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid token claims"}`))
				return
			}

			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				userID, _ = claims["user_id"].(string)
			}
			if userID == "" {
				userID = "dev-user-001"
			}

			username, _ := claims["username"].(string)
			if username == "" {
				username, _ = claims["name"].(string)
			}
			if username == "" {
				username = "user"
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UsernameKey, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the userID from context if set by Auth middleware.
func GetUserID(ctx context.Context) (string, bool) {
	val := ctx.Value(UserIDKey)
	if val == nil {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetUsername extracts the username from context if set by Auth middleware.
func GetUsername(ctx context.Context) (string, bool) {
	val := ctx.Value(UsernameKey)
	if val == nil {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}
