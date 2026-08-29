package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Processor func(ctx context.Context, job *Job) error

type WorkerPool struct {
	q          Queue
	rdb        *redis.Client
	processors map[string]Processor
}

func NewWorkerPool(q Queue, rdb *redis.Client) *WorkerPool {
	return &WorkerPool{
		q:          q,
		rdb:        rdb,
		processors: make(map[string]Processor),
	}
}

func (w *WorkerPool) Register(jobType string, p Processor) {
	w.processors[jobType] = p
}

func (w *WorkerPool) Start(ctx context.Context) {
	for jobType := range w.processors {
		go w.pollQueue(ctx, jobType)
	}
}

func (w *WorkerPool) pollQueue(ctx context.Context, jobType string) {
	queueKey := "queue:" + jobType
	p := w.processors[jobType]

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// BRPOP queueKey 2 seconds
			res, err := w.rdb.BRPop(ctx, 2*time.Second, queueKey).Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Worker polling error: %v", err)
					time.Sleep(1 * time.Second)
				}
				continue
			}

			if len(res) < 2 {
				continue
			}

			jobID := res[1]
			job, err := w.q.GetJob(ctx, jobID)
			if err != nil {
				log.Printf("Failed to get job %s: %v", jobID, err)
				continue
			}

			// Transition to running
			now := time.Now()
			job.Status = StatusRunning
			job.StartedAt = &now
			_ = w.q.UpdateJob(ctx, job)

			// Process
			err = p(ctx, job)
			
			nowDone := time.Now()
			job.DoneAt = &nowDone
			
			if err != nil {
				job.Status = StatusFailed
				job.Error = err.Error()
				_ = w.q.PublishEvent(ctx, job.UserID, "job.failed", job.ID, map[string]string{"error": err.Error()})
			} else {
				job.Status = StatusDone
				_ = w.q.PublishEvent(ctx, job.UserID, "job.completed", job.ID, job.Output)
			}
			_ = w.q.UpdateJob(ctx, job)
		}
	}
}
