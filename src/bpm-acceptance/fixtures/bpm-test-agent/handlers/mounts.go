package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func Mounts(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, string(data))
}
