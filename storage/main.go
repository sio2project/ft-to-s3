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
		return 0, utils.ErrorWrapper("Error while getting modified time", err)
	}
	if version <= dbModified {
		return dbModified, nil
	}
	logger.Debug("Version is greater than dbModified")

	oldFile, err := db.GetHashForPath(bucketName, path)
	if err != nil {
		return 0, utils.ErrorWrapper("Error while getting hash for path", err)
	}

	options := minio.PutObjectOptions{}
	var data bytes.Buffer
	var headersCalculated bool
	if sha256Digest == "" || logicalSize == -1 {
		logger.Debug("Calculating sha256Digest and logicalSize")
		headersCalculated = true
		var tempData []byte
		teeReader := io.TeeReader(reader, &data)
		if compressed {
			logger.Debug("File is compressed, reading gzip")
			tempData, err = utils.ReadGzip(teeReader)
			if err != nil {
				return 0, utils.ErrorWrapper("Error while reading gzip", err)
			}
		} else {
			logger.Debug("File is not compressed, reading data")
			tempData, err = io.ReadAll(teeReader)
			if err != nil {
				return 0, utils.ErrorWrapper("Error while reading data", err)
			}
		}

		sha256Digest = utils.Sha256Checksum(tempData)
		logicalSize = int64(len(tempData))
		logger.Debug("Calculated sha256Digest and logicalSize")
	}
	if compressed {
		logger.Debug("Setting ContentEncoding to gzip")
		options.ContentEncoding = "gzip"
		options.ContentType = "application/gzip"
	}

	refCount, err := db.GetRefCount(bucketName, sha256Digest)
	if err != nil {
		return 0, utils.ErrorWrapper("Error while getting refCount", err)
	}

	if refCount == 0 {
		logger.Debug("Storing with options", options)
		minioClient := GetClient()

		if headersCalculated {
			_, err = minioClient.PutObject(context.Background(), bucketName, sha256Digest, &data, size, options)
		} else {
			_, err = minioClient.PutObject(context.Background(), bucketName, sha256Digest, reader, size, options)
		}
		if err != nil {
			return 0, utils.ErrorWrapper("Error while putting object", err)
		}
	}

	logger.Info("Putting refFile")
	err = db.SetHashForPath(bucketName, path, sha256Digest)
	if err != nil {
		return 0, utils.ErrorWrapper("Error while setting hash for path", err)
	}

	logger.Info("Putting refCount")
	err = db.SetRefCount(bucketName, sha256Digest, refCount+1)
	if err != nil {
		return 0, utils.ErrorWrapper("Error while setting refCount", err)
	}

	err = db.SetModified(bucketName, path, version)
	if err != nil {
		return 0, utils.ErrorWrapper("Error while setting modified", err)
	}

	err = deleteByHash(bucketName, logger, oldFile, false)
	if err != nil {
		return 0, utils.ErrorWrapper("Error while deleting old file", err)
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

func GetList(bucketName string, logger *utils.LoggerObject, path string, last_modified int64) ([]string, error) {
	logger.Debug("storage.GetList called on", bucketName+":"+path)

	prefix := db.GetModifiedName(bucketName, path)
	keys, err := db.GetKeys(prefix)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, key := range keys {
		modified, err := db.GetModified(bucketName, key)
		if err != nil {
			return nil, err
		}
		if modified > last_modified {
			files = append(files, key)
		}
	}

	return files, nil
}
