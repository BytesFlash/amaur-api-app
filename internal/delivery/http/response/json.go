package response

import (
	"encoding/json"
	"net/http"

	"amaur/api/pkg/pagination"
)

type envelope map[string]any

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func OK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, envelope{"data": data})
}

func Created(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, envelope{"data": data})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Paginated(w http.ResponseWriter, items any, meta pagination.Meta) {
	writeJSON(w, http.StatusOK, envelope{
		"data": items,
		"meta": meta,
	})
}

func BadRequest(w http.ResponseWriter, code, message string) {
	writeJSON(w, http.StatusBadRequest, envelope{
		"error": ErrorBody{Code: code, Message: message},
	})
}

func Unauthorized(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusUnauthorized, envelope{
		"error": ErrorBody{Code: "UNAUTHORIZED", Message: message},
	})
}

func Forbidden(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusForbidden, envelope{
		"error": ErrorBody{Code: "FORBIDDEN", Message: message},
	})
}

func NotFound(w http.ResponseWriter, code, message string) {
	writeJSON(w, http.StatusNotFound, envelope{
		"error": ErrorBody{Code: code, Message: message},
	})
}

func Conflict(w http.ResponseWriter, code, message string) {
	writeJSON(w, http.StatusConflict, envelope{
		"error": ErrorBody{Code: code, Message: message},
	})
}

func InternalError(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, envelope{
		"error": ErrorBody{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"},
	})
}

func ValidationError(w http.ResponseWriter, details map[string]string) {
	writeJSON(w, http.StatusUnprocessableEntity, envelope{
		"error": envelope{
			"code":    "VALIDATION_ERROR",
			"message": "Validation failed",
			"details": details,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
