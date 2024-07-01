package storage

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/sio2project/ft-to-s3/v1/db"
	"github.com/sio2project/ft-to-s3/v1/utils"
	"io"
)

func Store(bucketName string, logger *utils.LoggerObject, path string, reader io.Reader, version int64, size int64,
	compressed bool, sha256Digest string, logicalSize int64) (int64, error) {
	logger.Debug("storage.Store called on", bucketName+":"+path)

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
	logger.Debug("Version is greater than dbModified")

	oldFile, err := db.GetHashForPath(bucketName, path)
	if err != nil {
		return 0, err
	}

	options := minio.PutObjectOptions{}
	var data bytes.Buffer
	if sha256Digest == "" || logicalSize == -1 {
		var tempData []byte
		teeReader := io.TeeReader(reader, &data)
		if compressed {
			tempData, err = utils.ReadGzip(teeReader)
			if err != nil {
				return 0, err
			}
		} else {
			tempData, err = io.ReadAll(teeReader)
			if err != nil {
				return 0, err
			}
		}

		sha256Digest = utils.Sha256Checksum(tempData)
		logicalSize = int64(len(tempData))
	}
	if compressed {
		logger.Debug("Setting ContentEncoding to gzip")
		options.ContentEncoding = "gzip"
		options.ContentType = "application/gzip"
	}

	refCount, err := db.GetRefCount(bucketName, sha256Digest)
	if err != nil {
		return 0, err
	}

	if refCount == 0 {
		logger.Debug("Storing with options", options)
		minioClient := GetClient()

		_, err = minioClient.PutObject(context.Background(), bucketName, sha256Digest, &data, size, options)
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

func Get(bucketName string, logger *utils.LoggerObject, path string) *GetResult {
	logger.Debug("storage.Get called on", bucketName+":"+path)

	fileHash, err := db.GetHashForPath(bucketName, path)
	if err != nil {
		return &GetResult{Err: err}
	}
	if fileHash == "" {
		return &GetResult{Found: false}
	}

	lastModified, err := db.GetModified(bucketName, path)
	if err != nil {
		return &GetResult{Err: err}
	}

	minioClient := GetClient()
	info, err := minioClient.StatObject(context.Background(), bucketName, fileHash, minio.StatObjectOptions{})
	if err != nil {
		minioErr := minio.ToErrorResponse(err)
		if minioErr.Code == "NoSuchKey" {
			return &GetResult{Found: false}
		}
		return &GetResult{Err: err}
	}
	gziped := info.ContentType == "application/gzip"

	reader, err := minioClient.GetObject(context.Background(), bucketName, fileHash, minio.GetObjectOptions{})
	if err != nil {
		minioErr := minio.ToErrorResponse(err)
		if minioErr.Code == "NoSuchKey" {
			return &GetResult{Found: false}
		}
		return &GetResult{Err: err}
	}

	return &GetResult{
		Found:        true,
		File:         reader,
		Gziped:       gziped,
		LastModified: lastModified,
		LogicalSize:  info.Size,
	}
}
