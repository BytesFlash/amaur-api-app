package middleware

import (
	"net/http"

	"amaur/api/internal/delivery/http/response"
)

// RequirePermission checks that the authenticated user has the given permission key (module:action).
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				response.Unauthorized(w, "Not authenticated")
				return
			}
			if claims.HasRole("super_admin") {
				next.ServeHTTP(w, r)
				return
			}
			if !claims.HasPermission(perm) {
				response.Forbidden(w, "You do not have permission to perform this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole checks that the authenticated user has at least one of the given roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				response.Unauthorized(w, "Not authenticated")
				return
			}
			for _, role := range roles {
				if claims.HasRole(role) {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Forbidden(w, "Insufficient role")
		})
	}
}

// IsCompanyScopedRole returns true when the user belongs to a company-scoped portal role.
func IsCompanyScopedRole(claims interface{ HasRole(string) bool }) bool {
	if claims == nil {
		return false
	}
	return claims.HasRole("company_hr") || claims.HasRole("company_worker")
}

func IsCompanyWorkerRole(claims interface{ HasRole(string) bool }) bool {
	if claims == nil {
		return false
	}
	return claims.HasRole("company_worker")
}

// IsPatientScopedRole returns true when the user must be constrained to its own patient_id scope.
func IsPatientScopedRole(claims interface{ HasRole(string) bool }) bool {
	if claims == nil {
		return false
	}
	return claims.HasRole("company_worker") || claims.HasRole("patient")
}
