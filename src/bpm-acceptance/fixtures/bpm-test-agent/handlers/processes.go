package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

func Processes(w http.ResponseWriter, r *http.Request) {
	items, err := filepath.Glob("/proc/[0-9]*")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		body, err := ioutil.ReadFile(filepath.Join(item, "cmdline"))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, fmt.Sprintf("%s %s", filepath.Base(item), string(body)))
	}
}
