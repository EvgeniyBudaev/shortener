package main

import (
	"net/http"
	"strings"
)

const httpProtocol = "http://"

var urls = make(map[string][]byte)

func mainPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(httpProtocol + req.Host + "/"))

		return
	} else if req.Method == http.MethodGet {
		reqPathElements := strings.Split(req.URL.Path, "/")
		id := reqPathElements[len(reqPathElements)-1]
		res.Header().Set("Location", string(urls[id]))
		res.WriteHeader(http.StatusTemporaryRedirect)

		return
	}

	res.WriteHeader(http.StatusBadRequest)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, mainPage)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
