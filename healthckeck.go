package labbot

import (
	"encoding/json"
	"net/http"
	"runtime"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Goroutine int `json:"goroutine"`
	}{
		Goroutine: runtime.NumGoroutine(),
	})
}
