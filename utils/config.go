package utils

import (
	"encoding/json"
	"io"
	"os"

	"github.com/sio2project/ft-to-s3/v1/db"
)

type Instance struct {
	Port       string `json:"port"`
	BucketName string `json:"bucketName"`
}

type Config struct {
	Instances []Instance     `json:"instances"`
	Redis     db.RedisConfig `json:"db"`
}

func LoadConfig(configPath string) *Config {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("Config file does not exist")
	}
	confFile, err := os.Open(configPath)
	if err != nil {
		panic(err)
	}
	defer confFile.Close()
	byteValue, _ := io.ReadAll(confFile)
	var config Config
	if err := json.Unmarshal(byteValue, &config); err != nil {
		panic(err)
	}
	return &config
}
