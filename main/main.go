package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mr-joshcrane/assistant"
	"github.com/mr-joshcrane/oracle"
)

func main() {

	log.Fatal(http.ListenAndServe("localhost:8082", mux))
}

