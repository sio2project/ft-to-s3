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
	defer fileMutex.Unlock(context.Background())

	dbModified, err := db.GetModified(bucketName, path)
	if err != nil {
		return 0, err
	}
	if version <= dbModified {
		return dbModified, nil
	}

	oldFile, err := db.GetHashForPath(bucketName, path)
	if err != nil {
		return 0, err
	}

	options := minio.PutObjectOptions{}
	if sha256Digest == "" || logicalSize == -1 {
		var data []byte
		if compressed {
			data, err = utils.ReadGzip(reader)
			if err != nil {
				return 0, err
			}
		} else {
			data, err = io.ReadAll(reader)
			if err != nil {
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
		return 0, err
	}

	if refCount == 0 {
		logger.Debug("Storing with options ", options)
		minioClient := GetClient()

		_, err = minioClient.PutObject(context.Background(), bucketName, sha256Digest, reader, size, options)
		if err != nil {
			return 0, err
		}
	}

	logger.Info("Putting refFile")
	err = db.SetHashForPath(bucketName, path, sha256Digest)
	if err != nil {
		return 0, err
	}

	logger.Info("Putting refCount")
	err = db.SetRefCount(bucketName, sha256Digest, refCount+1)
	if err != nil {
		return 0, err
	}

	err = db.SetModified(bucketName, path, version)
	if err != nil {
		return 0, err
	}

	err = deleteByHash(bucketName, logger, oldFile, false)
	if err != nil {
		return 0, err
	}

	return version, nil
}

func deleteByHash(bucketName string, logger *utils.LoggerObject, path string, lock bool) error {
	logger.Debug("DeleteByHash called on ", path)
	return nil
}
