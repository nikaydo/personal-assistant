package logg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/nikaydo/personal-assistant/internal/models"
)

var (
	TaskLevel     = -2
	AnswerLevel   = -1
	QuestionLevel = 1
	ErrorLevel    = 2
	MemoryLevel   = 3
	AgentLevel    = 5
	minLogLevel   = slog.Level(-10)
)

type teeWriter struct {
	console io.Writer
	file    io.Writer
}

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func (t *teeWriter) Write(p []byte) (int, error) {
	if _, err := t.console.Write(p); err != nil {
		return 0, err
	}

	clean := ansiRegexp.ReplaceAll(p, []byte{})
	if _, err := t.file.Write(clean); err != nil {
		return 0, err
	}

	return len(p), nil
}

type ColorHandler struct {
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
	group string
}

type Logger struct {
	Customlogger *slog.Logger
	Mode         string
}

func (l *Logger) WithModule(module string) *Logger {
	module = strings.ToUpper(strings.TrimSpace(module))
	if module == "" {
		return l
	}

	return &Logger{
		Customlogger: l.Customlogger.With(slog.String("module", module)),
		Mode:         l.Mode,
	}
}

func (l *Logger) Agent(msg ...any) {
	message, attrs := normalizeLogArgs(msg...)
	l.Customlogger.LogAttrs(context.Background(), slog.Level(AgentLevel), message, attrs...)
}

func (l *Logger) Task(msg ...any) {
	message, attrs := normalizeLogArgs(msg...)
	l.Customlogger.LogAttrs(context.Background(), slog.Level(TaskLevel), message, attrs...)
}

func (l *Logger) Memory(msg ...any) {
	message, attrs := normalizeLogArgs(msg...)
	l.Customlogger.LogAttrs(context.Background(), slog.Level(MemoryLevel), message, attrs...)
}

func (l *Logger) Answer(msg any) {
	message, attrs := normalizeLogArgs(msg)
	l.Customlogger.LogAttrs(context.Background(), slog.Level(AnswerLevel), message, attrs...)
}

func (l *Logger) Question(msg any) {
	message, attrs := normalizeLogArgs(msg)
	l.Customlogger.LogAttrs(context.Background(), slog.Level(QuestionLevel), message, attrs...)
}

func (l *Logger) Info(msg ...any) {
	message, attrs := normalizeLogArgs(msg...)
	l.Customlogger.LogAttrs(context.Background(), slog.LevelInfo, message, attrs...)
}

func (l *Logger) Warn(msg ...any) {
	message, attrs := normalizeLogArgs(msg...)
	l.Customlogger.LogAttrs(context.Background(), slog.LevelWarn, message, attrs...)
}

func (l *Logger) Debug(msg ...any) {
	message, attrs := normalizeLogArgs(msg...)
	l.Customlogger.LogAttrs(context.Background(), slog.LevelDebug, message, attrs...)
}

func (l *Logger) Error(msg ...any) {
	message, attrs := normalizeLogArgs(msg...)
	l.Customlogger.LogAttrs(context.Background(), slog.Level(ErrorLevel), message, attrs...)
}

func InitLogger() *Logger {
	return InitLoggerWithMode("")
}

func InitLoggerWithMode(mode string) *Logger {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "full"
	}
	logFileName := time.Now().Format("2006-01-02_15-04-05") + ".log"
	if _, err := os.ReadDir("./data/log"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir("data/logs", 0o755); err != nil {
				log.Println(err)
			}

		} else {
			log.Println("logs cant make directory", err)
		}
	}
	logFile, err := os.OpenFile(fmt.Sprintf("./data/logs/%s", logFileName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	var fileHandler slog.Handler
	if err == nil {
		fileHandler = NewPlainHandler(logFile, minLogLevel)
	}

	var consoleHandler slog.Handler
	switch mode {
	case "none":
		consoleHandler = NewPlainHandler(io.Discard, minLogLevel)
	case "pretty":
		consoleHandler = NewPrettyHandler(os.Stdout, minLogLevel)
	default:
		consoleHandler = NewColorHandler(os.Stdout, minLogLevel)
	}

	if fileHandler == nil {
		return &Logger{
			Customlogger: slog.New(consoleHandler),
			Mode:         mode,
		}
	}

	return &Logger{
		Customlogger: slog.New(NewMultiHandler(fileHandler, consoleHandler)),
		Mode:         mode,
	}
}

func NewColorHandler(w io.Writer, level slog.Level) *ColorHandler {
	return &ColorHandler{
		w:     w,
		level: level,
	}
}

func (h *ColorHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *ColorHandler) Handle(_ context.Context, r slog.Record) error {
	allAttrs := make([]slog.Attr, 0, len(h.attrs)+r.NumAttrs())
	allAttrs = append(allAttrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		allAttrs = append(allAttrs, a)
		return true
	})

	module := "SYSTEM"
	fields := make(map[string]any, len(allAttrs))
	level := levelName(r.Level)

	for _, a := range allAttrs {
		if a.Key == "module" {
			module = strings.ToUpper(strings.TrimSpace(fmt.Sprint(a.Value.Any())))
			continue
		}
		fields[a.Key] = a.Value.Any()
	}

	line := fmt.Sprintf(
		"%s %s %s %s",
		r.Time.Format("2006-01-02 15:04:05"),
		colorLevel(level, r.Level),
		colorModule(module),
		formatMessage(r.Message),
	)

	if len(fields) > 0 {
		line += " | " + formatFieldsJSON(fields)
	}

	_, err := io.WriteString(h.w, line+"\n")
	return err
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &ColorHandler{
		w:     h.w,
		level: h.level,
		attrs: newAttrs,
		group: h.group,
	}
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return &ColorHandler{
		w:     h.w,
		level: h.level,
		attrs: h.attrs,
		group: name,
	}
}

type PlainHandler struct {
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
	group string
}

func NewPlainHandler(w io.Writer, level slog.Level) *PlainHandler {
	return &PlainHandler{
		w:     w,
		level: level,
	}
}

func (h *PlainHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *PlainHandler) Handle(_ context.Context, r slog.Record) error {
	allAttrs := make([]slog.Attr, 0, len(h.attrs)+r.NumAttrs())
	allAttrs = append(allAttrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		allAttrs = append(allAttrs, a)
		return true
	})

	module := "SYSTEM"
	fields := make(map[string]any, len(allAttrs))
	level := levelName(r.Level)

	for _, a := range allAttrs {
		if a.Key == "module" {
			module = strings.ToUpper(strings.TrimSpace(fmt.Sprint(a.Value.Any())))
			continue
		}
		fields[a.Key] = a.Value.Any()
	}

	line := fmt.Sprintf(
		"%s %s [%s] %s",
		r.Time.Format("2006-01-02 15:04:05"),
		level,
		module,
		formatMessage(r.Message),
	)

	if len(fields) > 0 {
		line += " | " + formatFieldsJSON(fields)
	}

	_, err := io.WriteString(h.w, line+"\n")
	return err
}

func (h *PlainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &PlainHandler{
		w:     h.w,
		level: h.level,
		attrs: newAttrs,
		group: h.group,
	}
}

func (h *PlainHandler) WithGroup(name string) slog.Handler {
	return &PlainHandler{
		w:     h.w,
		level: h.level,
		attrs: h.attrs,
		group: name,
	}
}

type PrettyHandler struct {
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
	group string
}

func NewPrettyHandler(w io.Writer, level slog.Level) *PrettyHandler {
	return &PrettyHandler{
		w:     w,
		level: level,
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	allAttrs := make([]slog.Attr, 0, len(h.attrs)+r.NumAttrs())
	allAttrs = append(allAttrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		allAttrs = append(allAttrs, a)
		return true
	})

	module := "SYSTEM"
	attrs := make(map[string]any, len(allAttrs))
	for _, a := range allAttrs {
		if a.Key == "module" {
			module = strings.ToUpper(strings.TrimSpace(fmt.Sprint(a.Value.Any())))
			continue
		}
		attrs[a.Key] = a.Value.Any()
	}
	if module == "QUEUE" {
		return nil
	}

	timestamp := r.Time.Format("2006-01-02 15:04:05")
	switch {
	case r.Level == slog.Level(QuestionLevel):
		line := fmt.Sprintf(
			"%s %s %s Question: %s",
			timestamp,
			colorLevel(levelName(r.Level), r.Level),
			colorModule(module),
			prettyMessage(r.Message, attrs),
		)
		_, err := io.WriteString(h.w, line+"\n")
		return err
	case r.Level == slog.Level(AnswerLevel):
		line := fmt.Sprintf(
			"%s %s %s Answer: %s",
			timestamp,
			colorLevel(levelName(r.Level), r.Level),
			colorModule(module),
			extractAnswer(attrs, r.Message),
		)
		_, err := io.WriteString(h.w, line+"\n")
		return err
	case r.Level == slog.Level(AgentLevel) || r.Level == slog.Level(TaskLevel):
		label, text, ok := prettyAgentLine(attrs)
		if !ok {
			return nil
		}
		line := fmt.Sprintf(
			"%s %s %s %s: %s",
			timestamp,
			colorLevel(levelName(r.Level), r.Level),
			colorModule(module),
			label,
			text,
		)
		_, err := io.WriteString(h.w, line+"\n")
		return err
	case r.Level == slog.Level(MemoryLevel):
		return nil
	case r.Level == slog.LevelDebug:
		return nil
	}

	level := levelName(r.Level)
	line := fmt.Sprintf(
		"%s %s %s %s",
		timestamp,
		colorLevel(level, r.Level),
		colorModule(module),
		formatMessage(r.Message),
	)
	if errVal, ok := attrs["error"]; ok {
		line += " | error=" + formatValueForLog(errVal)
	}
	_, err := io.WriteString(h.w, line+"\n")
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &PrettyHandler{
		w:     h.w,
		level: h.level,
		attrs: newAttrs,
		group: h.group,
	}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{
		w:     h.w,
		level: h.level,
		attrs: h.attrs,
		group: name,
	}
}

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	out := make([]slog.Handler, 0, len(handlers))
	for _, h := range handlers {
		if h != nil {
			out = append(out, h)
		}
	}
	return &MultiHandler{handlers: out}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithAttrs(attrs))
	}
	return &MultiHandler{handlers: next}
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithGroup(name))
	}
	return &MultiHandler{handlers: next}
}

func levelName(level slog.Level) string {
	switch {
	case level == slog.LevelWarn:
		return "WARN"
	case level == slog.LevelInfo:
		return "INFO"
	case level == slog.Level(TaskLevel):
		return "TASK"
	case level == slog.Level(QuestionLevel):
		return "QUESTION"
	case level == slog.Level(AnswerLevel):
		return "ANSWER"
	case level == slog.Level(ErrorLevel):
		return "ERROR"
	case level == slog.Level(MemoryLevel):
		return "MEMORY"
	case level == slog.Level(AgentLevel):
		return "AGENT"
	default:
		return "DEBUG"
	}
}

func colorLevel(level string, slogLevel slog.Level) string {
	switch {
	case slogLevel == slog.LevelWarn:
		return "\x1b[33m" + level + "\x1b[0m"
	case slogLevel == slog.LevelInfo:
		return "\x1b[36m" + level + "\x1b[0m"
	case slogLevel == slog.Level(TaskLevel):
		return "\x1b[35m" + level + "\x1b[0m"
	case slogLevel == slog.Level(QuestionLevel):
		return "\x1b[34m" + level + "\x1b[0m"
	case slogLevel == slog.Level(AnswerLevel):
		return "\x1b[32m" + level + "\x1b[0m"
	case slogLevel == slog.Level(ErrorLevel):
		return "\x1b[31m" + level + "\x1b[0m"
	case slogLevel == slog.Level(MemoryLevel):
		return "\x1b[96m" + level + "\x1b[0m"
	case slogLevel == slog.Level(AgentLevel):
		return "\x1b[30m" + level + "\x1b[0m"
	default:
		return "\x1b[90m" + level + "\x1b[0m"
	}
}

func colorModule(module string) string {
	return "\x1b[1m[" + module + "]\x1b[0m"
}

func formatMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "-"
	}
	return msg
}

func formatFieldsJSON(fields map[string]any) string {
	data, err := json.Marshal(fields)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func formatValueForLog(v any) string {
	if v == nil {
		return "nil"
	}

	switch val := v.(type) {
	case string:
		return strconvQuote(val)
	case error:
		return strconvQuote(val.Error())
	}

	data, err := json.Marshal(v)
	if err == nil {
		return string(data)
	}

	return strconvQuote(fmt.Sprintf("%+v", v))
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func prettyMessage(msg string, attrs map[string]any) string {
	if msg != "" {
		return msg
	}
	if v, ok := attrs["data"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "-"
}

func extractAnswer(attrs map[string]any, fallback string) string {
	if fallback != "" {
		return fallback
	}
	keys := []string{"data", "data1", "data2"}
	for _, key := range keys {
		v, ok := attrs[key]
		if !ok {
			continue
		}
		switch val := v.(type) {
		case models.ResponseBody:
			if content := val.GetContent(); content != "" {
				return content
			}
		case *models.ResponseBody:
			if val != nil {
				if content := val.GetContent(); content != "" {
					return content
				}
			}
		case string:
			if val != "" {
				return val
			}
		}
	}
	return "-"
}

func prettyAgentLine(attrs map[string]any) (string, string, bool) {
	if v, ok := attrs["thought"]; ok {
		return "LLM Thought", fmt.Sprint(v), true
	}
	if v, ok := attrs["tool"]; ok {
		args := ""
		if av, ok := attrs["args"]; ok {
			args = fmt.Sprint(av)
		}
		if args != "" {
			return "Tool", fmt.Sprintf("%v args=%s", v, args), true
		}
		return "Tool", fmt.Sprint(v), true
	}
	return "", "", false
}

func normalizeLogArgs(args ...any) (string, []slog.Attr) {
	if len(args) == 0 {
		return "", nil
	}

	msg := ""
	start := 0
	if first, ok := args[0].(string); ok {
		msg = first
		start = 1
	}

	attrs := make([]slog.Attr, 0, len(args)/2+1)
	dataIndex := 1

	for i := start; i < len(args); {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok && isFieldKey(key) {
				attrs = append(attrs, slog.Any(key, args[i+1]))
				i += 2
				continue
			}
		}

		if err, ok := args[i].(error); ok {
			attrs = append(attrs, slog.String("error", err.Error()))
		} else {
			key := "data"
			if dataIndex > 1 || len(args[start:]) > 1 {
				key = fmt.Sprintf("data%d", dataIndex)
			}
			attrs = append(attrs, slog.Any(key, args[i]))
			dataIndex++
		}
		i++
	}

	return msg, attrs
}

func isFieldKey(s string) bool {
	s = strings.TrimSpace(s)
	return s != "" && !strings.Contains(s, " ")
}
