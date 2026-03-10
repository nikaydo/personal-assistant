package llmcalls

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

var askFn = Ask

var ErrQueueStopped = errors.New("queue is stopped")

type Queue struct {
	cfg config.Config
	log *logg.Logger

	jobs    chan queueJob
	initMu  sync.Mutex
	started atomic.Bool
	nextID  atomic.Int64
	cancel  context.CancelFunc
	ctx     context.Context
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
	q.ctx = workerCtx
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
	q.cancel = nil
	q.stopMu.Unlock()
	if cancel != nil {
		cancel()
	}
	q.stopWg.Wait()
	q.stopMu.Lock()
	q.ctx = nil
	q.stopMu.Unlock()
	q.started.Store(false)
}

// sanitizeMessages ensures no message will be marshalled without a content field.
// the OpenRouter API treats an omitted "content" as null, which triggers a 400 error
// when the field is expected to be a string.  We replace any empty string with a
// single space so the JSON encoder always emits `"content":" "`.
func sanitizeMessages(msgs []models.Message) []models.Message {
	for i, m := range msgs {
		if m.Content == "" {
			msgs[i].Content = " "
		}
	}
	return msgs
}

func (q *Queue) AddToQueue(item QueueItem) (models.ResponseBody, error) {
	if !q.started.Load() {
		return models.ResponseBody{}, errors.New("queue is not started")
	}
	ctx := q.getContext()
	if ctx == nil {
		return models.ResponseBody{}, ErrQueueStopped
	}

	// make sure the request body is safe: content must not be empty
	item.Body.Messages = sanitizeMessages(item.Body.Messages)

	item.ID = int(q.nextID.Add(1))
	job := queueJob{
		item:     item,
		resultCh: make(chan models.ResponseBody, 1),
	}
	select {
	case q.jobs <- job:
	case <-ctx.Done():
		return models.ResponseBody{}, fmt.Errorf("%w: %v", ErrQueueStopped, ctx.Err())
	}
	q.debug(
		"Queue item enqueued",
		"item_id", item.ID,
		"pending", len(q.jobs),
		"messages_count", len(item.Body.Messages),
		"model", item.Body.Model,
	)
	var resp models.ResponseBody
	select {
	case resp = <-job.resultCh:
	case <-ctx.Done():
		return models.ResponseBody{}, fmt.Errorf("%w: %v", ErrQueueStopped, ctx.Err())
	}
	if resp.Error.Message != "" {
		q.error("Queue item failed", "item_id", item.ID, "error", resp.Error.Message)
	} else {
		q.info("Queue item completed", "item_id", item.ID, "total_tokens", resp.Usage.TotalTokens, "model", resp.Model)
	}
	return resp, nil
}

func (q *Queue) process(job queueJob, ctx context.Context) {
	q.debug("Queue processing started", "item_id", job.item.ID)
	select {
	case <-ctx.Done():
		job.resultCh <- models.ResponseBody{Error: models.Error{Message: ErrQueueStopped.Error()}}
		return
	default:
	}

	resp, err := askFn(job.item.Body, q.cfg)
	if err != nil {
		resp = models.ResponseBody{Error: models.Error{Message: err.Error()}}
	}
	job.resultCh <- resp
}

func (q *Queue) getContext() context.Context {
	q.stopMu.Lock()
	defer q.stopMu.Unlock()
	return q.ctx
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
