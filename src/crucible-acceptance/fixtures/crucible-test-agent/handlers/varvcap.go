package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func ListVarVcap(w http.ResponseWriter, r *http.Request) {
	items, err := ioutil.ReadDir("/var/vcap")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		fmt.Fprintln(w, item.Name())
	}
}
