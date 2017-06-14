package handlers

import (
	"fmt"
	"net/http"
	"os/user"
)

func Whoami(w http.ResponseWriter, r *http.Request) {
	usr, err := user.Current()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, usr.Username)
}
