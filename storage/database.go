package storage

import (
	"github.com/Focinfi/sqs/models"
	"github.com/Focinfi/sqs/storage/etcd"
	"github.com/Focinfi/sqs/storage/gomap"
	"github.com/Focinfi/sqs/storage/memcached"
	"github.com/Focinfi/sqs/storage/redis"
)

var defaultKV models.KV
var defaultIncrementer models.Incrementer
var etcdIncrementer *etcd.Incrementer
var etcdKV *etcd.KV
var etcdWatcher *etcd.Watcher
var memcachedKV *memcached.KV
var redisPriorityList *redis.PriorityList

func init() {
	// gomap
	defaultKV = gomap.New()

	// etcd kv
	if kv, err := etcd.NewKV(); err != nil {
		panic(err)
	} else {
		etcdKV = kv
	}

	// etcd watcher
	etcdWatcher = etcd.NewWatcher(etcdKV)

	// etcd incrementer
	etcdIncrementer = etcd.NewIncrementer(etcdKV)
	defaultIncrementer = etcdIncrementer

	// memcahed
	if kv, err := memcached.New(); err != nil {
		panic(err)
	} else {
		memcachedKV = kv
	}

	// redis
	if pl, err := redis.New(); err != nil {
		panic(err)
	} else {
		redisPriorityList = pl
	}
}
