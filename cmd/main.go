package main

import (
	"log"
	"os"

	"github.com/mr-joshcrane/oracle"
	"github.com/mr-joshcrane/tldr"
)

func main() {
	key := os.Getenv("OPENAI_API_KEY")
	o := oracle.NewOracle(key)
	srv := tldr.NewTLDRServer(o, ":8082")
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
