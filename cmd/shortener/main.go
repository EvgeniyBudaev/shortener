package main

import (
	"net/http"
	"strings"
)

func mainPage(res http.ResponseWriter, req *http.Request) {
	host := req.Host
	path := req.URL.Path
	url := host + path
	id := strings.TrimPrefix(req.URL.Path, "/")

	if req.Method == http.MethodPost {
		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(url))
		return
	} else if id != "" && req.Method == http.MethodGet {
		res.Header().Set("Content-Type", "text/plain")
		res.Header().Add("Location", id)
		res.WriteHeader(http.StatusTemporaryRedirect)
		return
	} else {
		http.Error(res, "Bad Request", http.StatusBadRequest)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, mainPage)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
