package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func VarVcap(w http.ResponseWriter, r *http.Request) {
	items, err := ioutil.ReadDir("/var/vcap")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		fmt.Fprintln(w, item.Name())
	}
}

func VarVcapJobs(w http.ResponseWriter, r *http.Request) {
	items, err := ioutil.ReadDir("/var/vcap/jobs")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		fmt.Fprintln(w, item.Name())
	}
}

func VarVcapData(w http.ResponseWriter, r *http.Request) {
	items, err := ioutil.ReadDir("/var/vcap/data")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		fmt.Fprintln(w, item.Name())
	}
}
