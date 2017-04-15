package storage

import (
	"github.com/Focinfi/sqs/external"
	"github.com/Focinfi/sqs/models"
)

// Storage defines storage
type Storage struct {
	*Nodes
	*Queue
	*Message
	*Squad
}

// DefaultStorage default storage
var DefaultStorage = &Storage{}

func init() {
	DefaultStorage.Queue = &Queue{db: sqsMetaKV, store: DefaultStorage, inc: etcdIncrementer}
	DefaultStorage.Message = &Message{db: messageKV, store: DefaultStorage}
	DefaultStorage.Squad = &Squad{db: sqsMetaKV, store: DefaultStorage}
	DefaultStorage.Nodes = &Nodes{db: ClusterMetaKV, store: DefaultStorage}

	// TODO: move into db/seeds
	DefaultStorage.Queue.db.Put(models.QueueListKey(external.Root.ID()), "[]")
}
