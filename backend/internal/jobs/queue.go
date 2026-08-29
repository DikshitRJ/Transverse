package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	StatusQueued  = "queued"
	StatusRunning = "running"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

type Job struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	JobType   string          `json:"job_type"`
	Status    string          `json:"status"`
	InputRef  json.RawMessage `json:"input_ref,omitempty"`
	Output    json.RawMessage `json:"output_json,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	StartedAt *time.Time      `json:"started_at,omitempty"`
	DoneAt    *time.Time      `json:"completed_at,omitempty"`
}

type Queue interface {
	Enqueue(ctx context.Context, job *Job) error
	GetJob(ctx context.Context, jobID string) (*Job, error)
	UpdateJob(ctx context.Context, job *Job) error
	PublishEvent(ctx context.Context, userID string, eventType string, jobID string, data interface{}) error
}

type redisQueue struct {
	rdb *redis.Client
}

func NewQueue(rdb *redis.Client) Queue {
	return &redisQueue{rdb: rdb}
}

func (q *redisQueue) Enqueue(ctx context.Context, job *Job) error {
	job.Status = StatusQueued
	job.CreatedAt = time.Now()

	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	// Save job state
	if err := q.rdb.Set(ctx, "job:"+job.ID, data, 24*time.Hour).Err(); err != nil {
		return err
	}

	// Push to queue list for workers
	if err := q.rdb.LPush(ctx, "queue:"+job.JobType, job.ID).Err(); err != nil {
		return err
	}

	return nil
}

func (q *redisQueue) GetJob(ctx context.Context, jobID string) (*Job, error) {
	data, err := q.rdb.Get(ctx, "job:"+jobID).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job not found")
		}
		return nil, err
	}

	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (q *redisQueue) UpdateJob(ctx context.Context, job *Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.rdb.Set(ctx, "job:"+job.ID, data, 24*time.Hour).Err()
}

func (q *redisQueue) PublishEvent(ctx context.Context, userID string, eventType string, jobID string, data interface{}) error {
	event := map[string]interface{}{
		"type":   eventType,
		"job_id": jobID,
		"data":   data,
	}
	payload, _ := json.Marshal(event)
	channel := fmt.Sprintf("user:%s:events", userID)
	return q.rdb.Publish(ctx, channel, payload).Err()
}
