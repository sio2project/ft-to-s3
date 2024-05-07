package main

import (
	"flag"
	"log"

	"github.com/sio2project/ft-to-s3/v1/db"
	"github.com/sio2project/ft-to-s3/v1/proxy"
	"github.com/sio2project/ft-to-s3/v1/utils"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to the config file")
	flag.Parse()
	if configPath == "" {
		log.Fatal("Config file path is required")
	}
	config := utils.LoadConfig(configPath)

	utils.ConfigureLogging(&config.Logging)
	db.StartClient(&config.Redis)
	proxy.Start(config)

}
