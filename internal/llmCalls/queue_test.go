package llmcalls

import (
	"errors"
	"testing"
	"time"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func TestAddToQueue_ReturnsWhenQueueStops(t *testing.T) {
	oldAskFn := askFn
	started := make(chan struct{})
	release := make(chan struct{})
	askFn = func(body models.RequestBody, cfg config.Config) (models.ResponseBody, error) {
		close(started)
		<-release
		return models.ResponseBody{
			Model: body.Model,
			Usage: models.Usage{TotalTokens: 1},
		}, nil
	}
	t.Cleanup(func() {
		askFn = oldAskFn
	})

	q := NewQueue(config.Config{}, 1, nil)
	q.QueueStart()

	errCh := make(chan error, 1)
	go func() {
		_, err := q.AddToQueue(QueueItem{
			Body: models.RequestBody{
				Model: "test-model",
			},
		})
		errCh <- err
	}()

	<-started
	stopDone := make(chan struct{})
	go func() {
		q.Stop()
		close(stopDone)
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, ErrQueueStopped) {
			t.Fatalf("unexpected error: got=%v want=%v", err, ErrQueueStopped)
		}
	case <-time.After(time.Second):
		t.Fatalf("AddToQueue blocked after Stop")
	}

	close(release)

	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatalf("Stop blocked")
	}
}
