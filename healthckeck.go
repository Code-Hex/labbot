package labbot

import (
	"fmt"
	"net/http"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello")
}
