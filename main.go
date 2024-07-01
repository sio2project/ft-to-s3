package main

import (
	"github.com/sio2project/ft-to-s3/v1/migrate"
	"github.com/sio2project/ft-to-s3/v1/proxy"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("No arguments provided")
	}
	mode := os.Args[1]
	if mode == "server" {
		proxy.Main()
	} else if mode == "migrate" {
		migrate.Main()
	} else {
		log.Fatal("Invalid mode. First argument has to be one of: server, migrate")
	}
}
