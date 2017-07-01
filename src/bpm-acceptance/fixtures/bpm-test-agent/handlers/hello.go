package handlers

import (
	"fmt"
	"net/http"
	"os"
)

func Hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "BPM is %s!\n", os.Getenv("BPM"))
}
