package llmcalls

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

type Queue struct {
	cfg config.Config
	log *logg.Logger

	jobs    chan queueJob
	initMu  sync.Mutex
	started atomic.Bool
	nextID  atomic.Int64
	cancel  context.CancelFunc
	stopWg  sync.WaitGroup
	stopMu  sync.Mutex
}

type QueueItem struct {
	ID   int
	Body models.RequestBody
}

type queueJob struct {
	item     QueueItem
	resultCh chan models.ResponseBody
}

func NewQueue(cfg config.Config, buffer int, logger *logg.Logger) *Queue {
	q := &Queue{
		cfg: cfg,
		log: logger,
	}
	q.init(buffer)
	return q
}

func (q *Queue) init(buffer int) {
	q.initMu.Lock()
	defer q.initMu.Unlock()
	if q.jobs != nil {
		return
	}
	if buffer <= 0 {
		buffer = 64
	}
	q.jobs = make(chan queueJob, buffer)
	q.debug("Queue initialized", "buffer", buffer)
}

func (q *Queue) QueueStart() {
	q.QueueStartWithContext(context.Background())
}

func (q *Queue) QueueStartWithContext(ctx context.Context) {
	q.init(64)
	if !q.started.CompareAndSwap(false, true) {
		q.debug("QueueStart called on already started queue")
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	workerCtx, cancel := context.WithCancel(ctx)
	q.stopMu.Lock()
	q.cancel = cancel
	q.stopMu.Unlock()
	q.info("Queue worker started", "buffer_capacity", cap(q.jobs))

	q.stopWg.Go(func() {
		for {
			select {
			case <-workerCtx.Done():
				q.info("Queue worker stopped")
				return
			case job := <-q.jobs:
				q.process(job, workerCtx)
			}
		}
	})
}

func (q *Queue) Stop() {
	q.info("Queue stop requested")
	q.stopMu.Lock()
	cancel := q.cancel
	q.stopMu.Unlock()
	if cancel != nil {
		cancel()
	}
	q.stopWg.Wait()
	q.started.Store(false)
}

func (q *Queue) AddToQueue(item QueueItem) (models.ResponseBody, error) {
	if !q.started.Load() {
		return models.ResponseBody{}, errors.New("queue is not started")
	}
	item.ID = int(q.nextID.Add(1))
	job := queueJob{
		item:     item,
		resultCh: make(chan models.ResponseBody, 1),
	}
	q.jobs <- job
	q.debug(
		"Queue item enqueued",
		"item_id", item.ID,
		"pending", len(q.jobs),
		"messages_count", len(item.Body.Messages),
		"model", item.Body.Model,
	)
	resp := <-job.resultCh
	if resp.Error.Message != "" {
		q.error("Queue item failed", "item_id", item.ID, "error", resp.Error.Message)
	} else {
		q.info("Queue item completed", "item_id", item.ID, "total_tokens", resp.Usage.TotalTokens, "model", resp.Model)
	}
	return resp, nil
}

func (q *Queue) process(job queueJob, _ context.Context) {
	q.debug("Queue processing started", "item_id", job.item.ID)
	resp, err := Ask(job.item.Body, q.cfg)
	if err != nil {
		resp = models.ResponseBody{Error: models.Error{Message: err.Error()}}
	}
	job.resultCh <- resp
}

func (q *Queue) debug(msg ...any) {
	if q.log == nil {
		return
	}
	q.log.Debug(msg...)
}

func (q *Queue) info(msg ...any) {
	if q.log == nil {
		return
	}
	q.log.Info(msg...)
}

func (q *Queue) error(msg ...any) {
	if q.log == nil {
		return
	}
	q.log.Error(msg...)
}
