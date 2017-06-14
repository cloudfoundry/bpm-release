package handlers

import (
	"fmt"
	"net/http"
	"os"
)

func Hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Crucible is %s!\n", os.Getenv("CRUCIBLE"))
}
