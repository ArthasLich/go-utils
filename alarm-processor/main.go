package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	http.HandleFunc("/",func(w http.ResponseWriter, r *http.Request) {
		
	})
}
