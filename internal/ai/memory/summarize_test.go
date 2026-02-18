package memory

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/nikaydo/personal-assistant/internal/config"
)

func TestPlanSummaryRunByStep(t *testing.T) {
	mem := &Memory{
		Cfg: &config.Config{
			SummaryMemoryStep: 3,
		},
	}

	mem.FillShortMemory("q1", "a1")
	run, _ := mem.PlanSummaryRun()
	if run {
		t.Fatalf("expected no summary run on first message")
	}

	mem.FillShortMemory("q2", "a2")
	run, _ = mem.PlanSummaryRun()
	if run {
		t.Fatalf("expected no summary run on second message")
	}

	mem.FillShortMemory("q3", "a3")
	run, snapshot := mem.PlanSummaryRun()
	if !run {
		t.Fatalf("expected summary run on third message")
	}
	if got := len(snapshot); got != 3 {
		t.Fatalf("expected snapshot len 3, got %d", got)
	}
	if snapshot[0].UserQuestion != "q1" || snapshot[2].UserQuestion != "q3" {
		t.Fatalf("unexpected snapshot order: %#v", snapshot)
	}
	if got := mem.SummaryCounter(); got != 0 {
		t.Fatalf("expected counter reset to 0, got %d", got)
	}
}

func TestPlanSummaryRunConcurrentOnlyOneRunner(t *testing.T) {
	mem := &Memory{
		Cfg: &config.Config{
			SummaryMemoryStep: 1,
		},
	}
	mem.FillShortMemory("q1", "a1")

	var started int32
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if run, _ := mem.PlanSummaryRun(); run {
				atomic.AddInt32(&started, 1)
			}
		}()
	}

	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&started); got != 1 {
		t.Fatalf("expected exactly one runner, got %d", got)
	}
}

func TestPlanSummaryRunSnapshotImmutable(t *testing.T) {
	mem := &Memory{
		Cfg: &config.Config{
			SummaryMemoryStep: 1,
		},
	}
	mem.FillShortMemory("old-q", "old-a")
	run, snapshot := mem.PlanSummaryRun()
	if !run {
		t.Fatalf("expected summary run")
	}
	if len(snapshot) != 1 {
		t.Fatalf("expected snapshot len 1, got %d", len(snapshot))
	}

	mem.FillShortMemory("new-q", "new-a")
	if snapshot[0].UserQuestion != "old-q" || snapshot[0].LLMAnswer != "old-a" {
		t.Fatalf("snapshot changed after memory mutation: %#v", snapshot[0])
	}
}
