package storage

import (
	"bytes"
	"context"
	"io"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/sio2project/ft-to-s3/v1/db"
	"github.com/sio2project/ft-to-s3/v1/utils"
)

func getRefCountName(bucketName string, sha256sum string) string {
	return "ref_count:" + bucketName + ":" + sha256sum
}

func getRefFileName(bucketName string, path string) string {
	return "ref_file:" + bucketName + ":" + path
}

func getModifiedName(bucketName string, path string) string {
	return "modified:" + bucketName + ":" + path
}

func Store(bucketName string, logger *utils.LoggerObject, path string, reader io.Reader, version int64, size int64,
	compressed bool, sha256Digest string, logicalSize int64) (int64, error) {
	etcd := db.GetClient()
	session := db.GetSession(etcd)
	defer session.Close()
	defer etcd.Close()
	fileMutex := db.GetMutex(session, bucketName+":"+path)
	fileMutex.Lock(context.Background())

	resp, err := etcd.Get(etcd.Ctx(), getModifiedName(bucketName, path))
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}
	if resp.Count > 0 {
		dbModified, err := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
		if err != nil {
			fileMutex.Unlock(context.Background())
			return 0, err
		}
		if version <= dbModified {
			fileMutex.Unlock(context.Background())
			return dbModified, nil
		}
	}

	options := minio.PutObjectOptions{}
	if sha256Digest == "" || logicalSize == -1 {
		var data []byte
		if compressed {
			data, err = utils.ReadGzip(reader)
			if err != nil {
				fileMutex.Unlock(context.Background())
				return 0, err
			}
		} else {
			data, err = io.ReadAll(reader)
			if err != nil {
				fileMutex.Unlock(context.Background())
				return 0, err
			}
		}

		sha256Digest = utils.Sha256Checksum(data)
		logicalSize = int64(len(data))
		reader = bytes.NewReader(data)
	}
	if compressed {
		options.ContentEncoding = "gzip"
	}

	refCountName := getRefCountName(bucketName, sha256Digest)
	resp, err = etcd.Get(etcd.Ctx(), refCountName)
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}
	refCount := 0
	if resp.Count > 0 {
		refCount, err = strconv.Atoi(string(resp.Kvs[0].Value))
		if err != nil {
			fileMutex.Unlock(context.Background())
			return 0, err
		}
	}

	if refCount == 0 {
		logger.Info("Storing with options ", options)
		minio, err := GetClient()
		if err != nil {
			fileMutex.Unlock(context.Background())
			return 0, err
		}

		_, err = minio.PutObject(context.Background(), bucketName, sha256Digest, reader, size, options)
		if err != nil {
			fileMutex.Unlock(context.Background())
			return 0, err
		}
	}

	logger.Info("Putting refFile")
	refFile := getRefFileName(bucketName, path)
	_, err = etcd.Put(etcd.Ctx(), refFile, sha256Digest)
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}

	logger.Info("Putting refCount")
	modified := getModifiedName(bucketName, path)
	_, err = etcd.Put(etcd.Ctx(), modified, strconv.FormatInt(version, 10))
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}

	_, err = etcd.Put(etcd.Ctx(), refCountName, strconv.Itoa(refCount+1))
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}

	fileMutex.Unlock(context.Background())
	return version, nil
}
