package proxy

import (
	"flag"
	"github.com/sio2project/ft-to-s3/v1/db"
	"github.com/sio2project/ft-to-s3/v1/utils"
	"os"
)

func Main() {
	flagSet := flag.NewFlagSet("server", flag.ExitOnError)
	var configPath string
	flagSet.StringVar(&configPath, "config", "", "Path to the config file")
	err := flagSet.Parse(os.Args[2:])
	if err != nil {
		panic(err)
	}
	if configPath == "" {
		panic("Config file path is required")
	}
	config := utils.LoadConfig(configPath)

	utils.ConfigureLogging(&config.Logging)
	db.Configure(&config.Etcd)
	Start(config)
}
