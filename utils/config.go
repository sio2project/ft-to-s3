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

type MinioConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secret"`
	UseSSL          bool   `json:"useSSL"`
}

type LoggingConfig struct {
	Level string `json:"level"`
	File  string `json:"file"`
}

type Config struct {
	Instances []Instance    `json:"instances"`
	Etcd      EtcdConfig    `json:"etcd"`
	Minio     MinioConfig   `json:"minio"`
	Logging   LoggingConfig `json:"logging"`
}

func LoadConfig(configPath string) *Config {
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
