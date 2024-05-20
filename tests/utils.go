package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/sio2project/ft-to-s3/v1/db"
	"github.com/sio2project/ft-to-s3/v1/proxy"
	"github.com/sio2project/ft-to-s3/v1/storage"
	"github.com/sio2project/ft-to-s3/v1/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func setup() {
	config := utils.Config{
		Instances: []utils.Instance{
			{
				BucketName: "test",
				Port:       ":8080",
			},
		},
		Etcd: utils.EtcdConfig{
			Endpoints:   []string{"http://localhost:2379"},
			DialTimeout: 5,
			SessionTTL:  5,
		},
		Minio: utils.MinioConfig{
			Endpoint:        "localhost:9000",
			AccessKeyID:     "minio",
			SecretAccessKey: "minio123",
			UseSSL:          false,
		},
		Logging: utils.LoggingConfig{
			Level: "debug",
		},
	}
	utils.ConfigureLogging(&config.Logging)
	db.Configure(&config.Etcd)
	storage.Configure(&config)
	go proxy.Start(&config)
}

func destroy() {
	proxy.Stop()
}

func clean() {
	db.MustDelete("ref_count:", clientv3.WithPrefix())
	db.MustDelete("ref_file:", clientv3.WithPrefix())
	db.MustDelete("modified:", clientv3.WithPrefix())

	minioClient := storage.GetClient()
	buckets, err := minioClient.ListBuckets(context.Background())
	if err != nil {
		panic(err)
	}
	for _, bucket := range buckets {
		objects := minioClient.ListObjects(context.Background(), bucket.Name, minio.ListObjectsOptions{})
		for object := range objects {
			err = minioClient.RemoveObject(context.Background(), bucket.Name, object.Key, minio.RemoveObjectOptions{})
			if err != nil {
				panic(err)
			}
		}
	}
}

func makeRequest(urlString string, method string, body []byte, getParams map[string]string,
	headers map[string]string) (int, string) {
	if len(getParams) > 0 {
		urlString += "?"
		for key, value := range getParams {
			urlString += key + "=" + url.QueryEscape(value) + "&"
		}
		urlString = strings.TrimSuffix(urlString, "&")
	}
	req, err := http.NewRequest(method, urlString, bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return resp.StatusCode, string(respBody)
}
