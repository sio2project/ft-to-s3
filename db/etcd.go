package db

import (
	"strconv"
	"time"

	"github.com/sio2project/ft-to-s3/v1/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var etcdConfig *utils.EtcdConfig

// TODO: Check if etcd client can be used concurrently
var etcdClient *clientv3.Client

func Configure(config *utils.EtcdConfig) {
	etcdConfig = config
	etcdClient = getClient()
}

func getClient() *clientv3.Client {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdConfig.Endpoints,
		DialTimeout: time.Duration(etcdConfig.DialTimeout) * time.Second,
	})
	if err != nil {
		panic(err)
	}
	return client
}

func GetSession() *concurrency.Session {
	session, err := concurrency.NewSession(etcdClient, concurrency.WithTTL(etcdConfig.SessionTTL))
	if err != nil {
		panic(err)
	}
	return session
}

func GetMutex(session *concurrency.Session, key string) *concurrency.Mutex {
	mutex := concurrency.NewMutex(session, key)
	return mutex
}

func GetModified(bucketName string, path string) (int64, error) {
	resp, err := etcdClient.Get(etcdClient.Ctx(), getModifiedName(bucketName, path))
	if err != nil {
		return 0, err
	}
	if resp.Count == 0 {
		return 0, nil
	}
	return strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
}

func SetModified(bucketName string, path string, modified int64) error {
	_, err := etcdClient.Put(etcdClient.Ctx(), getModifiedName(bucketName, path), strconv.FormatInt(modified, 10))
	return err
}

func GetRefCount(bucketName string, sha256Digest string) (int, error) {
	resp, err := etcdClient.Get(etcdClient.Ctx(), getRefCountName(bucketName, sha256Digest))
	if err != nil {
		return 0, err
	}
	if resp.Count == 0 {
		return 0, nil
	}
	return strconv.Atoi(string(resp.Kvs[0].Value))
}

func SetRefCount(bucketName string, sha256Digest string, count int) error {
	_, err := etcdClient.Put(etcdClient.Ctx(), getRefCountName(bucketName, sha256Digest), strconv.Itoa(count))
	return err
}

func IncrementRefCount(bucketName string, sha256Digest string) error {
	refCount, err := GetRefCount(bucketName, sha256Digest)
	if err != nil {
		return err
	}
	return SetRefCount(bucketName, sha256Digest, refCount+1)
}

func GetHashForPath(bucketName string, path string) (string, error) {
	resp, err := etcdClient.Get(etcdClient.Ctx(), getRefFileName(bucketName, path))
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", nil
	}
	return string(resp.Kvs[0].Value), nil
}

func SetHashForPath(bucketName string, path string, sha256Digest string) error {
	_, err := etcdClient.Put(etcdClient.Ctx(), getRefFileName(bucketName, path), sha256Digest)
	return err
}

func Delete(key string, opts ...clientv3.OpOption) error {
	_, err := etcdClient.Delete(etcdClient.Ctx(), key, opts...)
	return err
}

func MustDelete(key string, opts ...clientv3.OpOption) {
	err := Delete(key, opts...)
	if err != nil {
		panic(err)
	}
}
