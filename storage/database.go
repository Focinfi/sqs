package storage

import (
	"fmt"

	"github.com/Focinfi/gosqs/config"
	"github.com/Focinfi/gosqs/models"
	"github.com/Focinfi/gosqs/storage/etcd"
	"github.com/Focinfi/gosqs/storage/gomap"
	"github.com/Focinfi/gosqs/storage/oncekv"
)

// ClusterMetaKV for nodes cluster
var ClusterMetaKV models.KV
var sqsMetaKV models.KV
var messageKV models.KV
var sqsMetaIncrementer models.Incrementer

// specific backend
var mapKV *gomap.KV
var mapIncrementer *gomap.Incrementer
var onceKV *oncekv.KV
var etcdKV *etcd.KV
var etcdIncrementer *etcd.Incrementer

func getEtcdKV() *etcd.KV {
	if etcdKV == nil {
		if kv, err := etcd.NewKV(); err != nil {
			panic(err)
		} else {
			etcdKV = kv
		}
	}

	return etcdKV
}

func newEtcdIncrementer() *etcd.Incrementer {
	if etcdIncrementer == nil {
		etcdIncrementer = etcd.NewIncrementer(etcdKV)
	}

	return etcdIncrementer
}

func getOnceKV() *oncekv.KV {
	if onceKV == nil {
		if kv, err := oncekv.NewKV(); err != nil {
			panic(err)
		} else {
			onceKV = kv
		}
	}

	return onceKV
}

func init() {
	//mapKV
	mapKV = gomap.New()
	mapIncrementer = gomap.NewIncrementer(mapKV)

	if config.Config.Env.IsProduction() {
		ClusterMetaKV = getEtcdKV()
		sqsMetaKV = getEtcdKV()
		sqsMetaIncrementer = newEtcdIncrementer()
		messageKV = getOnceKV()
	} else if config.Config.Env.IsDevelop() {
		ClusterMetaKV = getEtcdKV()
		sqsMetaKV = getEtcdKV()
		sqsMetaIncrementer = newEtcdIncrementer()
		messageKV = mapKV
	} else if config.Config.Env.IsTest() {
		ClusterMetaKV = mapKV
		sqsMetaKV = mapKV
		sqsMetaIncrementer = mapIncrementer
		messageKV = mapKV
	} else {
		panic(fmt.Sprintf("env '%s' is not allowed", config.Config.Env))
	}
}
