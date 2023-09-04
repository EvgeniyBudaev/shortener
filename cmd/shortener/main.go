package main

import (
	"crypto/rand"
	"io"
	"log"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
)

const httpProtocol = "http://"

func generateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}

func shortURL(ptrUrls *map[string][]byte) func(c *gin.Context) {
	return func(c *gin.Context) {
		req := c.Request
		res := c.Writer
		urls := *ptrUrls
		defer req.Body.Close()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.Fatal(err)
		}

		id, err := generateRandomString(8)
		if err != nil {
			log.Fatal(err)
		}
		urls[id] = body

		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(httpProtocol + req.Host + "/" + id))
	}
}

func redirectURL(ptrUrls *map[string][]byte) func(c *gin.Context) {
	return func(c *gin.Context) {
		res := c.Writer
		urls := *ptrUrls
		id := c.Param("id")
		originalURL := urls[id]
		if originalURL == nil {
			res.WriteHeader(http.StatusNotFound)
			return
		}

		res.Header().Set("Location", string(originalURL))
		res.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func setupRouter(ptrUrls *map[string][]byte) *gin.Engine {
	urls := *ptrUrls
	r := gin.Default()

	r.GET("/:id", redirectURL(&urls))
	r.POST("/", shortURL(&urls))

	return r
}

func main() {
	var urls = make(map[string][]byte)

	r := setupRouter(&urls)
	log.Fatal(r.Run())
}
