package storage

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sio2project/ft-to-s3/v1/utils"
)

var minioConfig *utils.MinioConfig

func Configure(config *utils.Config) {
	minioConfig = &config.Minio
	createBuckets(config)
}

func GetClient() (*minio.Client, error) {
	return minio.New(minioConfig.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioConfig.AccessKeyID, minioConfig.SecretAccessKey, ""),
		Secure: minioConfig.UseSSL,
	})
}

func createBuckets(config *utils.Config) {
	client, err := GetClient()
	if err != nil {
		panic(err)
	}
	for _, instance := range config.Instances {
		exists, err := client.BucketExists(context.Background(), instance.BucketName)
		if err != nil {
			panic(err)
		}
		if !exists {
			utils.MainLogger.Info("Creating bucket", instance.BucketName)
			err = client.MakeBucket(context.Background(), instance.BucketName, minio.MakeBucketOptions{})
			if err != nil {
				panic(err)
			}
		}
	}
}
