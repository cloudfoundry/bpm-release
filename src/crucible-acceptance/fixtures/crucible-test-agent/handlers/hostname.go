package handlers

import (
	"fmt"
	"net/http"
	"os"
)

func Hostname(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, hostname)
}
