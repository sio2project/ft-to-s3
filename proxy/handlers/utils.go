package handlers

import "net/http"

func set_content_type_json(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}
