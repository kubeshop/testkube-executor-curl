package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
func (r *MapRepository) Get(ctx context.Context, id string) (testkube.Execution, error) {
	v, ok := r.data.Load(id)
	if !ok {
		return testkube.Execution{}, fmt.Errorf("no execution with the id %s", id)
	}

	return v.(testkube.Execution), nil
}

// Insert inserts new execution result
func (r *MapRepository) Insert(ctx context.Context, result testkube.Execution) error {
	r.data.Store(result.Id, result)
	return nil
}

// Update updates execution result
func (r *MapRepository) Update(ctx context.Context, result testkube.Execution) error {
	return r.Insert(ctx, result)
}

// Update updates execution result
func (r *MapRepository) UpdateResult(ctx context.Context, id string, result testkube.ExecutionResult) error {
	v, ok := r.data.Load(id)
	if !ok {
		return fmt.Errorf("no execution with the id %s", id)
	}

	execution := v.(testkube.Execution)
	execution.ExecutionResult = &result

	return r.Insert(ctx, execution)
}

// QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
func (r *MapRepository) QueuePull(ctx context.Context) (testkube.Execution, error) {
	var id string
	var execution testkube.Execution
	// get a random execution
	r.data.Range(func(key, value interface{}) bool {
		id = key.(string)
		execution = value.(testkube.Execution)
		//when false is returned range function will exit,
		//the queued execution is needed so false is returned when the execution has status queued
		return *execution.ExecutionResult.Status != testkube.SUCCESS_ExecutionStatus
	})
	if len(id) == 0 || !execution.ExecutionResult.IsQueued() {
		return execution, mongo.ErrNoDocuments
	}
	execution.ExecutionResult.Status = testkube.StatusPtr(testkube.PENDING_ExecutionStatus)
	r.data.Store(id, execution)
	return execution, nil
}
