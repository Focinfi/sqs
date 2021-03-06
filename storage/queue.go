package storage

import (
	"encoding/json"

	"github.com/Focinfi/gosqs/config"
	"github.com/Focinfi/gosqs/errors"
	"github.com/Focinfi/gosqs/models"
	"github.com/Focinfi/gosqs/util/strconvutil"
)

// Queue stores data
type Queue struct {
	store *Storage
	db    models.KV
	inc   models.Incrementer
}

// All returns queue map for userID
func (s *Queue) All(userID int64) ([]models.Queue, error) {
	all := []models.Queue{}
	key := models.QueueListKey(userID)

	val, findErr := s.db.Get(key)
	if findErr == errors.DataNotFound {
		return nil, errors.UserNotFound
	}

	if findErr != nil {
		return nil, findErr
	}

	if val == "" {
		return all, nil
	}

	if err := json.Unmarshal([]byte(val), &all); err != nil {
		return nil, errors.DataBroken(key, err)
	}

	return all, nil
}

// One returns a queue for the userID with the queueName
func (s *Queue) One(userID int64, queueName string) (*models.Queue, error) {
	all, err := s.All(userID)
	if err != nil {
		return nil, err
	}

	if len(all) > config.Config.MaxQueueCountPerUser {
		return nil, errors.CanNotCreateMoreQueue
	}

	var theQueue *models.Queue
	for _, queue := range all {
		if queue.Name == queueName {
			*theQueue = queue
		}
	}

	if theQueue == nil {
		return nil, errors.QueueNotFound
	}

	return theQueue, nil
}

// Add add q for userID
func (s *Queue) Add(q *models.Queue) error {
	all, err := s.All(q.UserID)
	if err != nil {
		return err
	}

	for _, queue := range all {
		if queue.Name == q.Name {
			return errors.DuplicateQueue
		}
	}

	newAll := append(all, *q)
	data, err := json.Marshal(newAll)
	if err != nil {
		return errors.NewInternalErrorf(err.Error())
	}

	err = s.db.Put(models.QueueListKey(q.UserID), string(data))
	if err != nil {
		return errors.NewInternalErrorf(err.Error())
	}

	return nil
}

// Remove removes the queue with the name
func (s *Queue) Remove(userID int64, queueName string) error {
	all, err := s.All(userID)
	if err != nil {
		return err
	}

	index := -1
	for i, queue := range all {
		if queue.Name == queueName {
			index = i
		}
	}

	if index < 0 {
		return errors.QueueNotFound
	}

	all = append(all[:index], all[index+1:]...)
	data, err := json.Marshal(all)
	if err != nil {
		return errors.NewInternalErrorf(err.Error())
	}

	err = s.db.Put(models.QueueListKey(userID), string(data))
	if err != nil {
		return errors.NewInternalErrorf(err.Error())
	}

	return nil
}

// ApplyMessageIDRange try to apply message id range
func (s *Queue) ApplyMessageIDRange(userID int64, queueName string, size int) (int64, error) {
	key := models.QueueMaxIDKey(userID, queueName)
	return s.inc.Increment(key, size)
}

// InitMessageMaxID set queue max id, only if it havent been set
func (s *Queue) InitMessageMaxID(userID int64, queueName string, id int64) error {
	_, err := s.MessageMaxID(userID, queueName)
	if err == errors.DataNotFound {
		key := models.QueueMaxIDKey(userID, queueName)
		return s.db.Put(key, strconvutil.Int64toa(id))
	}

	if err != nil {
		return err
	}
	return nil
}

// MessageMaxID get the max id for the queue
func (s *Queue) MessageMaxID(userID int64, queueName string) (int64, error) {
	key := models.QueueMaxIDKey(userID, queueName)
	val, getErr := s.db.Get(key)
	if getErr != nil {
		return -1, getErr
	}

	maxID, err := strconvutil.ParseInt64(val)
	if err != nil {
		return -1, errors.DataBroken(key, err)
	}

	return maxID, nil
}
