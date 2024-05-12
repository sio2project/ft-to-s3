package db

import (
	"github.com/sio2project/ft-to-s3/v1/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"

	"time"
)

var etcdConfig *utils.EtcdConfig

func Configure(config *utils.EtcdConfig) {
	etcdConfig = config
}

func GetClient() *clientv3.Client {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdConfig.Endpoints,
		DialTimeout: time.Duration(etcdConfig.DialTimeout) * time.Second,
	})
	if err != nil {
		panic(err)
	}
	return client
}

func GetSession(client *clientv3.Client) *concurrency.Session {
	session, err := concurrency.NewSession(client, concurrency.WithTTL(etcdConfig.SessionTTL))
	if err != nil {
		panic(err)
	}
	return session
}

func GetMutex(session *concurrency.Session, key string) *concurrency.Mutex {
	mutex := concurrency.NewMutex(session, key)
	return mutex
}
