package memory

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/models"
	"github.com/pinecone-io/go-pinecone/v5/pinecone"
)

func (mem *Memory) PlanSummaryRun() (bool, []ShortTerm) {
	mem.mu.Lock()
	defer mem.mu.Unlock()

	step := mem.Cfg.SummaryMemoryStep
	if step <= 0 {
		return false, nil
	}

	mem.Count++
	if mem.Logger != nil {
		mem.Logger.Memory("SummMemory check", "counter", mem.Count, "step", step, "short_memory_len", len(mem.ShortTerm), "running", mem.summaryRunning)
	}

	if mem.Count < step {
		if mem.Logger != nil {
			mem.Logger.Memory("SummMemory skipped", "next_counter", mem.Count)
		}
		return false, nil
	}
	if mem.summaryRunning {
		mem.Count = step
		if mem.Logger != nil {
			mem.Logger.Memory("SummMemory skip: already running", "counter", mem.Count)
		}
		return false, nil
	}

	mem.Count = 0
	mem.summaryRunning = true

	snapshotLen := min(len(mem.ShortTerm), step)
	if snapshotLen == 0 {
		mem.summaryRunning = false
		return false, nil
	}

	snapshot := make([]ShortTerm, snapshotLen)
	copy(snapshot, mem.ShortTerm[len(mem.ShortTerm)-snapshotLen:])
	return true, snapshot
}

func (mem *Memory) SummMemoryFromSnapshot(snapshot []ShortTerm) error {
	if len(snapshot) == 0 {
		return fmt.Errorf("SummMemoryFromSnapshot: empty snapshot")
	}
	msg := mem.collectShortMemory(len(snapshot), mem.Cfg.PromtMemorySummary, snapshot)
	if mem.Logger != nil {
		mem.Logger.Memory("SummMemory started", "messages", len(msg))
	}
	b := models.RequestBody{
		Model:    mem.Cfg.ModelOpenRouter[0],
		Models:   mem.Cfg.ModelOpenRouter[1:],
		Messages: msg,
		Tools:    GetToolSummarize(),
		ToolsChoise: models.ToolsChoise{
			Type: "function",
			Function: models.ToolsChoisePayload{
				Name: "summarize",
			},
		},
	}
	resp, err := llmcalls.Ask(b, *mem.Cfg)
	if err != nil {
		if mem.Logger != nil {
			mem.Logger.Error("SummMemory ask failed:", err)
		}
		return err
	}
	if len(resp.Choices) == 0 {
		return fmt.Errorf("SummMemory: empty choices")
	}
	if len(resp.Choices[0].Message.ToolCalls) == 0 {
		return fmt.Errorf("SummMemory: empty tool_calls")
	}
	args := pinecone.IntegratedRecord{
		"id": uuid.New().String(),
	}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.ToolCalls[0].Function.Arguments), &args); err != nil {
		if mem.Logger != nil {
			mem.Logger.Error("SummMemory tool args parse failed:", err)
		}
		return err
	}
	var summary []*pinecone.IntegratedRecord
	if err := mem.DBase.Upsert(append(summary, &args)); err != nil {
		if mem.Logger != nil {
			mem.Logger.Error("SummMemory upsert failed:", err)
		}
		return err
	}
	if mem.Logger != nil {
		mem.Logger.Memory("SummMemory upsert success", "record_id", args["id"])
	}
	return nil
}

func (mem *Memory) FinishSummaryRun() {
	mem.mu.Lock()
	mem.summaryRunning = false
	mem.mu.Unlock()
}

func (mem *Memory) CheckLimits() {
	mem.mu.Lock()
	defer mem.mu.Unlock()
	f := len(mem.ShortTerm)
	if f >= 100 {
		mem.ShortTerm = mem.ShortTerm[1:]
	}
}

func GetToolSummarize() []models.Tool {
	return []models.Tool{
		{
			Type: "function",
			Function: models.Function{
				Name:        "summarize",
				Description: "A function for creating a short summary of a text based on its category, purpose, and importance.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"text": map[string]any{
							"type":        "string",
							"description": "Summarized text according to the history of previous queries",
						},
						"category": map[string]any{
							"type":        "string",
							"description": "Text category, such as 'news', 'report', 'technical document'. Used for contextual processing.",
						},
						"goal": map[string]any{
							"type":        "string",
							"description": "The purpose of the summary: for example, 'quick overview', 'report preparation', 'presentation creation'.",
						},
						"importance": map[string]any{
							"type":        "string",
							"description": "The importance level of the summary: 'high', 'medium', 'low'. Used to prioritize key information.",
						},
						"status": map[string]any{
							"type":        "string",
							"description": "The status of the summary: 'completed', 'in progress', 'failed'.",
						},
					},
					"required": []string{"text", "category", "goal", "importance", "status"},
				},
			},
		},
	}
}
