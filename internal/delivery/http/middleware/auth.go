package middleware

import (
	"context"
	"net/http"
	"strings"

	jwtpkg "amaur/api/pkg/jwt"
	"amaur/api/internal/delivery/http/response"
)

type contextKey string

const ClaimsKey contextKey = "claims"

func Authenticate(jwt *jwtpkg.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				response.Unauthorized(w, "Missing or invalid authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := jwt.ParseAccessToken(tokenStr)
			if err != nil {
				response.Unauthorized(w, "Invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) *jwtpkg.Claims {
	c, _ := ctx.Value(ClaimsKey).(*jwtpkg.Claims)
	return c
}
