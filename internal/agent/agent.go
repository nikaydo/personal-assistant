package agent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/shlex"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
	"github.com/nikaydo/personal-assistant/internal/services"
	command "github.com/nikaydo/personal-assistant/internal/services/command"
)

type Agent struct {
	Steps int
	Model string

	Dbase *database.Database

	Cfg config.Config

	Queue *llmcalls.Queue

	Logger *logg.Logger

	SystemPrompt string

	History *[]models.Message
}

type History struct {
	Type string
	Tool models.ToolsHistory
	Msg  models.Message
}

type AgentResponse struct {
	Thought  string `json:"thought"`
	Question string `json:"question,omitempty"`
	Func     Func   `json:"action"`
}

type Func struct {
	Function  string          `json:"function"`
	Mode      string          `json:"mode,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Args      json.RawMessage `json:"args,omitempty"`
}

func rawProvided(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && string(trimmed) != "null"
}

func normalizeActionFunction(functionName string, hasPayload bool) (string, error) {
	fn := strings.ToLower(strings.TrimSpace(functionName))
	if fn == "" {
		if hasPayload {
			return "command", nil
		}
		return "", nil
	}
	if strings.HasSuffix(fn, ".command") {
		return "command", nil
	}
	switch fn {
	case "command", "stop", "reasoning":
		return fn, nil
	default:
		return "", fmt.Errorf("unsupported action function: %s", functionName)
	}
}

func normalizeCommandSpec(resp AgentResponse) (command.CommandSpec, error) {
	hasPayload := rawProvided(resp.Func.Arguments) || rawProvided(resp.Func.Args)
	fn, err := normalizeActionFunction(resp.Func.Function, hasPayload)
	if err != nil {
		return command.CommandSpec{}, err
	}
	if fn != "command" {
		return command.CommandSpec{}, fmt.Errorf("unsupported action function: %s", resp.Func.Function)
	}
	if rawProvided(resp.Func.Arguments) && rawProvided(resp.Func.Args) {
		return command.CommandSpec{}, errors.New("conflicting action payload: both arguments and args are provided")
	}
	mode := strings.ToLower(strings.TrimSpace(resp.Func.Mode))
	if mode == "" {
		mode = "exec"
	}
	parseSpec := func(raw json.RawMessage) (command.CommandSpec, error) {
		raw = bytes.TrimSpace(raw)
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			s = strings.TrimSpace(s)
			if s == "" {
				return command.CommandSpec{}, errors.New("empty command string")
			}
			if mode == "shell" {
				return command.CommandSpec{Mode: "shell", Command: s}, nil
			}
			parts, err := shlex.Split(s)
			if err != nil {
				return command.CommandSpec{}, err
			}
			if len(parts) == 0 {
				return command.CommandSpec{}, errors.New("empty command string")
			}
			return command.CommandSpec{Mode: "exec", Command: parts[0], Args: parts[1:]}, nil
		}
		var arr []string
		if err := json.Unmarshal(raw, &arr); err == nil {
			if len(arr) == 0 {
				return command.CommandSpec{}, errors.New("empty command array")
			}
			if mode == "shell" {
				if len(arr) == 1 && strings.TrimSpace(arr[0]) != "" {
					return command.CommandSpec{Mode: "shell", Command: strings.TrimSpace(arr[0])}, nil
				}
				if len(arr) >= 3 && strings.EqualFold(arr[0], "sh") && arr[1] == "-c" {
					return command.CommandSpec{Mode: "shell", Command: arr[2]}, nil
				}
				return command.CommandSpec{}, errors.New("invalid shell array payload: use [\"<script>\"] or [\"sh\",\"-c\",\"<script>\"]")
			}
			return command.CommandSpec{Mode: "exec", Command: arr[0], Args: arr[1:]}, nil
		}
		var obj struct {
			Command string          `json:"command"`
			Args    []string        `json:"args"`
			Mode    string          `json:"mode"`
			Script  string          `json:"script"`
			Shell   json.RawMessage `json:"shell"`
		}
		if err := json.Unmarshal(raw, &obj); err == nil {
			localMode := mode
			if strings.TrimSpace(obj.Mode) != "" {
				localMode = strings.ToLower(strings.TrimSpace(obj.Mode))
			}
			if localMode == "shell" {
				if strings.TrimSpace(obj.Script) != "" {
					return command.CommandSpec{Mode: "shell", Command: strings.TrimSpace(obj.Script)}, nil
				}
				if strings.EqualFold(obj.Command, "sh") && len(obj.Args) >= 2 && obj.Args[0] == "-c" {
					return command.CommandSpec{Mode: "shell", Command: obj.Args[1]}, nil
				}
				return command.CommandSpec{}, errors.New("invalid shell payload: provide script or sh -c <script>")
			}
			if strings.TrimSpace(obj.Command) == "" {
				return command.CommandSpec{}, errors.New("missing command in action object")
			}
			return command.CommandSpec{Mode: "exec", Command: strings.TrimSpace(obj.Command), Args: obj.Args}, nil
		}
		return command.CommandSpec{}, errors.New("invalid action payload format")
	}
	var source json.RawMessage
	switch {
	case rawProvided(resp.Func.Arguments):
		source = resp.Func.Arguments
	case rawProvided(resp.Func.Args):
		source = resp.Func.Args
	default:
		return command.CommandSpec{}, errors.New("command action missing payload")
	}
	spec, err := parseSpec(source)
	if err != nil {
		return command.CommandSpec{}, err
	}
	if strings.TrimSpace(spec.Mode) == "" {
		spec.Mode = mode
	}
	if strings.TrimSpace(spec.Mode) == "" {
		spec.Mode = "exec"
	}
	if strings.TrimSpace(spec.Command) == "" {
		return command.CommandSpec{}, errors.New("command action resolved to empty command")
	}
	return spec, nil
}

func marshalCommandSpec(spec command.CommandSpec) string {
	b, _ := json.Marshal(spec)
	return string(b)
}

func normalizeRawCommandCall(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var obj struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Mode    string   `json:"mode"`
	}
	if err := json.Unmarshal([]byte(trimmed), &obj); err == nil && strings.TrimSpace(obj.Command) != "" {
		mode := strings.ToLower(strings.TrimSpace(obj.Mode))
		if mode == "" {
			mode = "exec"
		}
		return marshalCommandSpec(command.CommandSpec{
			Command: strings.TrimSpace(obj.Command),
			Args:    obj.Args,
			Mode:    mode,
		})
	}
	parts, err := shlex.Split(trimmed)
	if err != nil || len(parts) == 0 {
		return raw
	}
	return marshalCommandSpec(command.CommandSpec{
		Command: parts[0],
		Args:    parts[1:],
		Mode:    "exec",
	})
}

func toolResultFromOutput(output string) (command.ToolResult, bool) {
	var r command.ToolResult
	if err := json.Unmarshal([]byte(output), &r); err != nil {
		return command.ToolResult{}, false
	}
	return r, true
}

func failureSignature(toolName, normalizedArgs string, result command.ToolResult) (string, bool) {
	if result.Ok {
		return "", false
	}
	key := fmt.Sprintf("%s|%s|%d|%s", toolName, normalizedArgs, result.ExitCode, result.Error)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:]), true
}

type failureGuard struct {
	limit     int
	lastSig   string
	lastError string
	count     int
}

func newFailureGuard(limit int) *failureGuard {
	if limit <= 0 {
		limit = 2
	}
	return &failureGuard{limit: limit}
}

func (g *failureGuard) observe(sig, errMsg string, failed bool) bool {
	if !failed || sig == "" {
		g.lastSig = ""
		g.lastError = ""
		g.count = 0
		return false
	}
	if sig == g.lastSig {
		g.count++
	} else {
		g.lastSig = sig
		g.count = 1
	}
	g.lastError = errMsg
	return g.count >= g.limit
}

func parseAgentResponse(body models.ResponseBody) (AgentResponse, error) {
	var args AgentResponse
	err := json.Unmarshal([]byte(body.Choices[0].Message.ToolCalls[0].Function.Arguments), &args)
	if err != nil {
		return AgentResponse{}, err
	}
	return args, nil
}

func (a *Agent) Run(body models.ResponseBody) (models.ResponseBody, error) {
	if len(body.Choices) == 0 || len(body.Choices[0].Message.ToolCalls) == 0 {
		return models.ResponseBody{}, errors.New("agent: empty tool calls")
	}
	a.Logger.Agent("Agent.Run called", "initial_tool", body.Choices[0].Message.ToolCalls[0].Function.Name)
	r, err := parseAgentResponse(body)
	if err != nil {
		a.Logger.Error("parseAgentResponse failed", "error", err)
		return models.ResponseBody{}, err
	}
	if r.Question != "" {
		*a.History = append(*a.History, models.Message{Role: "user", Content: r.Question})
	}
	// make sure the first thought is never empty (omitempty in struct would drop it)
	safeThought := r.Thought
	if safeThought == "" {
		safeThought = " "
	}
	*a.History = append(*a.History, models.Message{
		Role:    "assistant",
		Content: safeThought})

	guard := newFailureGuard(2)

	// run a fixed number of steps rather than iterating over an integer value
	for i := range a.Steps {
		a.Logger.Agent("agent iteration", "step", i)
		respLLM, err := a.AskLLM("auto")
		if err != nil {
			a.Logger.Error("AskLLM failed", "error", err)
			return models.ResponseBody{}, err
		}
		a.Logger.Agent("AskLLM: ", respLLM)
		out, stop, err := a.RunTool(respLLM)
		if err != nil {
			a.Logger.Error("RunTool failed", "error", err)
			return models.ResponseBody{}, err
		}

		a.Logger.Agent("RunTool: ", out)
		if stop {
			a.Logger.Agent("agent stopping")
			if len(respLLM.Choices) > 0 {
				respLLM.Choices[0].Message.Content = out
				respLLM.Choices[0].Message.ToolCalls = nil
				respLLM.Choices[0].FinishReason = "stop"
			}
			return respLLM, nil
		}

		if tr, ok := toolResultFromOutput(out); ok {
			normalizedArgs := ""
			toolName := ""
			if len(respLLM.Choices) > 0 && len(respLLM.Choices[0].Message.ToolCalls) > 0 {
				tc := respLLM.Choices[0].Message.ToolCalls[0]
				toolName = tc.Function.Name
				switch tc.Function.Name {
				case "reasoning":
					args, err := parseAgentResponse(respLLM)
					if err == nil && args.Func.Function == "command" {
						if spec, err := normalizeCommandSpec(args); err == nil {
							normalizedArgs = marshalCommandSpec(spec)
							toolName = args.Func.Function
						}
					}
				case "command":
					normalizedArgs = normalizeRawCommandCall(tc.Function.Arguments)
				}
			}
			if sig, failed := failureSignature(toolName, normalizedArgs, tr); guard.observe(sig, tr.Error, failed) {
				msg := "Не удалось выполнить действие: инструмент возвращает одинаковую ошибку несколько раз. Попробуйте переформулировать задачу или указать другой способ выполнения."
				if strings.TrimSpace(guard.lastError) != "" {
					msg = fmt.Sprintf("%s Последняя ошибка: %s", msg, guard.lastError)
				}
				if len(respLLM.Choices) > 0 {
					respLLM.Choices[0].Message.Content = msg
					respLLM.Choices[0].Message.ToolCalls = nil
					respLLM.Choices[0].FinishReason = "stop"
				}
				return respLLM, nil
			}
		}

		if err := a.CollectContext(respLLM, out); err != nil {
			a.Logger.Error("CollectContext failed", "error", err)
		}
		// log history snapshot for debugging loops
		a.Logger.Agent("History after step", "count", len(*a.History))
	}
	return models.ResponseBody{}, errors.New("limit of steps")
}

func (a *Agent) RunTool(body models.ResponseBody) (string, bool, error) {
	if len(body.Choices) == 0 || len(body.Choices[0].Message.ToolCalls) == 0 {
		if body.Choices[0].FinishReason == "stop" {
			return body.Choices[0].Message.Content, true, nil
		}
		return "", false, errors.New("no tool calls")
	}
	if len(body.Choices[0].Message.ToolCalls) > 1 {
		a.Logger.Warn("multiple tool calls received; only the first will be executed")
	}
	i := body.Choices[0].Message.ToolCalls[0]
	switch i.Function.Name {
	case "reasoning":
		args, err := parseAgentResponse(body)
		if err != nil {
			return "", false, err
		}
		if args.Thought != "" {
			a.Logger.Agent("agent thought", "thought", args.Thought)
		}
		hasActionPayload := rawProvided(args.Func.Arguments) || rawProvided(args.Func.Args)
		if strings.TrimSpace(args.Func.Function) == "" && !hasActionPayload {
			return "", false, nil
		}
		spec, err := normalizeCommandSpec(args)
		if err != nil {
			data := command.ToolResult{
				Ok:        false,
				ExitCode:  -1,
				Error:     err.Error(),
				Retryable: true,
			}
			b, mErr := json.Marshal(data)
			if mErr != nil {
				return "", false, mErr
			}
			return string(b), false, nil
		}
		normalizedArgs := marshalCommandSpec(spec)
		a.Logger.Agent("agent tool", "tool", args.Func.Function, "args", normalizedArgs)
		svc := command.NewService()
		data := svc.ExecuteSpec(spec, services.CommandList{Type: false})
		if !data.Ok {
			a.Logger.Warn("command execution failed", "error", data.Error, "exit_code", data.ExitCode)
		}
		b, err := json.Marshal(data)
		if err != nil {
			return "", false, err
		}
		return string(b), false, nil
	case "command":
		a.Logger.Agent("agent tool", "tool", i.Function.Name, "args", i.Function.Arguments)
		svc := command.NewService()
		data := svc.ExecuteFromLLM(i.Function.Arguments, services.CommandList{Type: false})
		if !data.Ok {
			a.Logger.Warn("command execution failed", "error", data.Error, "exit_code", data.ExitCode)
		}
		b, err := json.Marshal(data)
		if err != nil {
			return "", false, err
		}
		return string(b), false, nil
	case "stop":
		var args struct {
			R string `json:"response"`
		}
		if err := json.Unmarshal([]byte(i.Function.Arguments), &args); err != nil {
			return "", false, err
		}
		*a.History = []models.Message{}
		return args.R, true, nil
	}
	return "", false, errors.New("unknown tool")
}

func getFuncNameArgs(body models.ResponseBody) (struct {
	Name         string
	Args         string
	FinishReason string
}, error) {
	var data struct {
		Name         string
		Args         string
		FinishReason string
	}
	if len(body.Choices) == 0 || len(body.Choices[0].Message.ToolCalls) == 0 {
		return data, errors.New("missing tool calls")
	}
	data.Name = body.Choices[0].Message.ToolCalls[0].Function.Name

	data.Args = body.Choices[0].Message.ToolCalls[0].Function.Arguments

	data.FinishReason = body.Choices[0].FinishReason
	return data, nil
}

func (a *Agent) CollectContext(body models.ResponseBody, funcOutput string) error {
	data, err := getFuncNameArgs(body)
	if err != nil {
		a.Logger.Error("CollectContext getFuncNameArgs failed", "error", err, "body", body)
		return err
	}
	tool := body.Choices[0].Message.ToolCalls[0]

	if data.Name == "reasoning" {
		args, err := parseAgentResponse(body)
		if err != nil {
			a.Logger.Error("CollectContext parseAgentResponse failed", "error", err)
			return err
		}
		if args.Func.Function != "" {
			actionArgs := ""
			if args.Func.Function == "command" {
				spec, err := normalizeCommandSpec(args)
				if err != nil {
					a.Logger.Error("CollectContext normalizeCommandSpec failed", "error", err)
					actionArgs = string(args.Func.Arguments)
				} else {
					actionArgs = marshalCommandSpec(spec)
				}
			} else {
				actionArgs = string(args.Func.Arguments)
			}
			if actionArgs == "null" {
				actionArgs = ""
			}
			content := fmt.Sprintf("args: %s\noutput: %s", actionArgs, funcOutput)
			*a.History = append(*a.History,
				models.Message{
					Role:      "function",
					ID:        tool.ID,
					Type:      tool.Type,
					Name:      args.Func.Function,
					Arguments: actionArgs,
					Output:    funcOutput,
					Content:   content,
				},
			)
		}
		// always append the assistant thought, substituting a space if it's empty
		thought := args.Thought
		if thought == "" {
			thought = " "
		}
		*a.History = append(*a.History,
			models.Message{
				Role:    "assistant",
				Content: thought,
			},
		)
		return nil
	}

	// non-reasoning tool: append tool output only
	argsForHistory := data.Args
	if data.Name == "command" {
		argsForHistory = normalizeRawCommandCall(data.Args)
	}
	content := fmt.Sprintf("args: %s\noutput: %s", argsForHistory, funcOutput)
	*a.History = append(*a.History,
		models.Message{
			Role:      "function",
			ID:        tool.ID,
			Type:      tool.Type,
			Name:      data.Name,
			Arguments: argsForHistory,
			Output:    funcOutput,
			Content:   content,
		},
	)

	return nil
}

// AskLLM forwards the current agent history to the LLM queue.  Before the
// request is enqueued we ensure that the optional system prompt (if set) is
// present as the first message; this lets the agent operate under a special
// instruction when running in "agent mode".
func (a *Agent) AskLLM(ToolsChoise string) (models.ResponseBody, error) {
	// insert system prompt at the beginning of the history if necessary
	if a.SystemPrompt != "" {
		if len(*a.History) == 0 || (*a.History)[0].Role != "system" || (*a.History)[0].Content != a.SystemPrompt {
			// prepend a copy so we don't mutate the original slice header
			sys := models.Message{Role: "system", Content: a.SystemPrompt}
			*a.History = append([]models.Message{sys}, *a.History...)
		}
	}

	respLLM, err := a.Queue.AddToQueue(llmcalls.QueueItem{Body: models.RequestBody{
		Model:       a.Model,
		Messages:    *a.History,
		ToolsChoise: ToolsChoise,
		Tools:       GetAgentTool(),
	}})
	if err != nil {
		return models.ResponseBody{}, err
	}
	return respLLM, nil
}
