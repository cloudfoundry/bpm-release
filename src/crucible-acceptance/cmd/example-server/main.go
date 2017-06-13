package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var port = flag.Int("port", -1, "port the server listens on")

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Crucible is %s!\n", os.Getenv("CRUCIBLE"))
}

func main() {
	flag.Parse()
	if *port == -1 {
		log.Fatal("no explicit port specified")
	}

	crucibleVar := os.Getenv("CRUCIBLE")
	if crucibleVar == "" {
		log.Fatal("Crucible environment variable not set")
	}

	http.HandleFunc("/", handler)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
