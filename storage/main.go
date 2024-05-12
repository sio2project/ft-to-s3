package storage

import (
	"github.com/sio2project/ft-to-s3/v1/db"
	"github.com/sio2project/ft-to-s3/v1/utils"

	"context"
	"io"
	"strconv"
)

func Store(bucketName string, logger *utils.LoggerObject, path string, reader io.Reader, version int64, size int64,
	compressed bool, sha256Digest string, logicalSize int64) (int64, error) {
	etcd := db.GetClient()
	session := db.GetSession(etcd)
	defer session.Close()
	defer etcd.Close()
	fileMutex := db.GetMutex(session, bucketName+":"+path)
	fileMutex.Lock(context.Background())

	resp, err := etcd.Get(etcd.Ctx(), "modified:"+bucketName+":"+path)
	if err != nil {
		logger.Error("Error", err)
		fileMutex.Unlock(context.Background())
		return 0, err
	}
	if resp.Count > 0 {
		dbModified, err := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
		if err != nil {
			logger.Error("Error", err)
			fileMutex.Unlock(context.Background())
			return 0, err
		}
		if version <= dbModified {
			fileMutex.Unlock(context.Background())
			return dbModified, nil
		}
	}

	if sha256Digest == "" || logicalSize == -1 {
		if compressed {

		} else {

		}
	}

	fileMutex.Unlock(context.Background())
	return 0, nil
}
