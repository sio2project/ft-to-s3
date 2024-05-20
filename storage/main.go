package storage

import (
	"bytes"
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/sio2project/ft-to-s3/v1/db"
	"github.com/sio2project/ft-to-s3/v1/utils"
)

func Store(bucketName string, logger *utils.LoggerObject, path string, reader io.Reader, version int64, size int64,
	compressed bool, sha256Digest string, logicalSize int64) (int64, error) {
	session := db.GetSession()
	defer session.Close()
	fileMutex := db.GetMutex(session, bucketName+":"+path)
	fileMutex.Lock(context.Background())

	dbModified, err := db.GetModified(bucketName, path)
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}
	if version <= dbModified {
		fileMutex.Unlock(context.Background())
		return dbModified, nil
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

	refCount, err := db.GetRefCount(bucketName, sha256Digest)
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}

	if refCount == 0 {
		logger.Debug("Storing with options ", options)
		minioClient := GetClient()

		_, err = minioClient.PutObject(context.Background(), bucketName, sha256Digest, reader, size, options)
		if err != nil {
			fileMutex.Unlock(context.Background())
			return 0, err
		}
	}

	logger.Info("Putting refFile")
	err = db.SetHashForPath(bucketName, path, sha256Digest)
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}

	logger.Info("Putting refCount")
	err = db.SetRefCount(bucketName, sha256Digest, refCount+1)
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}

	err = db.SetModified(bucketName, path, version)
	if err != nil {
		fileMutex.Unlock(context.Background())
		return 0, err
	}

	fileMutex.Unlock(context.Background())
	return version, nil
}
