package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"go.mongodb.org/mongo-driver/mongo"
)

type MapRepository struct {
	data   sync.Map
	queued sync.Map
}

func NewMapRepository() *MapRepository {
	return &MapRepository{}
}

// Get gets execution result by id
func (r *MapRepository) Get(ctx context.Context, id string) (kubtest.Execution, error) {
	v, ok := r.data.Load(id)
	if ok {
		return v.(kubtest.Execution), nil
	}

	v, ok = r.queued.Load(id)
	if ok {
		return v.(kubtest.Execution), nil
	}

	return kubtest.Execution{}, fmt.Errorf("No execution with the id %s", id)
}

// Insert inserts new execution result
func (r *MapRepository) Insert(ctx context.Context, result kubtest.Execution) error {
	if result.IsQueued() {
		r.queued.Store(result.Id, result)
	} else {
		r.data.Store(result.Id, result)
	}
	return nil
}

// Update updates execution result
func (r *MapRepository) Update(ctx context.Context, result kubtest.Execution) error {
	return r.Insert(ctx, result)
}

// QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
func (r *MapRepository) QueuePull(ctx context.Context) (kubtest.Execution, error) {
	var id string
	var execution kubtest.Execution
	// get a random execution
	r.queued.Range(func(key, value interface{}) bool {
		id = key.(string)
		execution = value.(kubtest.Execution)
		return false
	})
	if len(id) == 0 {
		return execution, mongo.ErrNoDocuments
	}
	r.queued.Delete(id)
	return execution, nil
}
