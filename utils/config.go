package utils

import (
	"encoding/json"
	"io"
	"os"
)

type Instance struct {
	Port       string `json:"port"`
	BucketName string `json:"bucketName"`
}

type RedisConfig struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type EtcdConfig struct {
	Endpoints   []string `json:"endpoints"`
	DialTimeout int      `json:"dialTimeout"`
	SessionTTL  int      `json:"sessionTTL"`
}

type LoggingConfig struct {
	Level string `json:"level"`
	File  string `json:"file"`
}

type Config struct {
	Instances []Instance    `json:"instances"`
	Etcd      EtcdConfig    `json:"etcd"`
	Logging   LoggingConfig `json:"logging"`
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
