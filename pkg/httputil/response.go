package httputil

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with the given status code
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// Error writes a JSON error response
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

// Success wraps data in a standard response format
func Success(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, map[string]interface{}{"data": data})
}

// Paginated writes a paginated JSON response
func Paginated(w http.ResponseWriter, data interface{}, total int64, page, limit int32) {
	JSON(w, http.StatusOK, map[string]interface{}{
		"data":  data,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
