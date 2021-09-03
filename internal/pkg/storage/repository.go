package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"go.mongodb.org/mongo-driver/mongo"
)

type MapRepository struct {
	data sync.Map
}

// NewMapRepository creates a MapRepository
func NewMapRepository() *MapRepository {
	return &MapRepository{}
}

// Get gets execution result by id
func (r *MapRepository) Get(ctx context.Context, id string) (kubtest.Execution, error) {
	v, ok := r.data.Load(id)
	if !ok {
		return kubtest.Execution{}, fmt.Errorf("No execution with the id %s", id)
	}

	return v.(kubtest.Execution), nil
}

// Insert inserts new execution result
func (r *MapRepository) Insert(ctx context.Context, result kubtest.Execution) error {
	r.data.Store(result.Id, result)
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
	r.data.Range(func(key, value interface{}) bool {
		id = key.(string)
		execution = value.(kubtest.Execution)
		//when false is returned range function will exit,
		//the queued execution is needed so false is returned when the execution has status queued
		return execution.Status != kubtest.ExecutionStatusQueued
	})
	if len(id) == 0 || !execution.IsQueued() {
		return execution, mongo.ErrNoDocuments
	}
	execution.Status = kubtest.ExecutionStatusPending
	r.data.Store(id, execution)
	return execution, nil
}
