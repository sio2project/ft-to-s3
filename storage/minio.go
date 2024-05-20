package storage

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sio2project/ft-to-s3/v1/utils"
)

var minioConfig *utils.MinioConfig

// TODO: Check if minio client can be used concurrently
var minioClient *minio.Client

func Configure(config *utils.Config) {
	minioConfig = &config.Minio
	var err error
	minioClient, err = getClient()
	if err != nil {
		panic(err)
	}
	createBuckets(config)
}

func getClient() (*minio.Client, error) {
	return minio.New(minioConfig.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioConfig.AccessKeyID, minioConfig.SecretAccessKey, ""),
		Secure: minioConfig.UseSSL,
	})
}

func GetClient() *minio.Client {
	return minioClient
}

func createBuckets(config *utils.Config) {
	for _, instance := range config.Instances {
		exists, err := minioClient.BucketExists(context.Background(), instance.BucketName)
		if err != nil {
			panic(err)
		}
		if !exists {
			utils.MainLogger.Info("Creating bucket", instance.BucketName)
			err = minioClient.MakeBucket(context.Background(), instance.BucketName, minio.MakeBucketOptions{})
			if err != nil {
				panic(err)
			}
		}
	}
}
