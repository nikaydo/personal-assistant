package logg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"github.com/fatih/color"
)

var (
	fieldColor = color.New(color.FgCyan, color.Bold)
	typeColor  = color.New(color.FgHiBlack)
	nilColor   = color.New(color.FgHiBlack)

	TaskLevel     = -2
	AnswerLevel   = -1
	QuestionLevel = 1
	ErrorLevel    = 2
	MemoryLevel   = 3
)

type ColorHandler struct {
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
	group string
}

type Logger struct {
	Customlogger *slog.Logger
}

func (l *Logger) Task(t string, msg ...any) {
	l.Customlogger.LogAttrs(context.Background(), slog.Level(TaskLevel), ColorStruct(msg), slog.String("type", t))
}

func (l *Logger) Memory(msg ...any) {
	l.Customlogger.LogAttrs(context.Background(), slog.Level(MemoryLevel), ColorStruct(msg))
}

func (l *Logger) Answer(msg any) {
	l.Customlogger.LogAttrs(context.Background(), slog.Level(AnswerLevel), ColorStruct(msg))
}

func (l *Logger) Question(msg any) {
	l.Customlogger.LogAttrs(context.Background(), slog.Level(QuestionLevel), ColorStruct(msg))
}

func (l *Logger) Info(msg ...any) {
	var parts []string
	for _, m := range msg {
		parts = append(parts, ColorStruct(m))
	}
	l.Customlogger.LogAttrs(context.Background(), slog.LevelInfo, ColorStruct(strings.Join(parts, " ")))
}

func (l *Logger) Error(msg any) {
	l.Customlogger.LogAttrs(context.Background(), slog.Level(ErrorLevel), ColorStruct(msg))
}

func InitLogger() *Logger {
	return &Logger{
		Customlogger: slog.New(NewColorHandler(os.Stdout, slog.LevelDebug)),
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
	level := colorizeLevel(r.Level)

	// время
	timeStr := color.New(color.FgCyan).Sprint(
		r.Time.Format("2006-01-02 15:04:05"),
	)

	// сообщение
	msg := r.Message

	// атрибуты
	attrs := ""
	r.Attrs(func(a slog.Attr) bool {
		attrs += a.Key + "=" + a.Value.String() + " "
		return true
	})

	_, err := io.WriteString(h.w, timeStr+" "+level+" "+msg+" "+attrs+"\n")
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

func colorizeLevel(level slog.Level) string {
	switch {
	case level == slog.LevelWarn:
		return color.New(color.FgYellow, color.Bold).Sprint("WARN ")
	case level == slog.LevelInfo:
		return color.New(color.FgGreen).Sprint("INFO ")
	case level == slog.Level(TaskLevel):
		return color.New(color.BgMagenta).Sprint("TASK")
	case level == slog.Level(QuestionLevel):
		return color.New(color.FgHiBlue).Sprint("QUESTION ")
	case level == slog.Level(AnswerLevel):
		return color.New(color.FgHiGreen).Sprint("ANSWER ")
	case level == slog.Level(ErrorLevel):
		return color.New(color.FgHiRed).Sprint("ERROR ")
	case level == slog.Level(MemoryLevel):
		return color.New(color.FgYellow).Sprint("MEMORY ")
	default:
		return color.New(color.FgHiBlack).Sprint("DEBUG")
	}
}

func ColorStruct(v any) string {
	return formatValue(reflect.ValueOf(v))
}

func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return nilColor.Sprint("nil")
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nilColor.Sprint("nil")
		}
		return formatValue(v.Elem())
	}

	switch v.Kind() {

	case reflect.Struct:
		return formatStruct(v)

	case reflect.String:
		return color.BlackString("%s", v.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return color.YellowString("%d", v.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return color.YellowString("%d", v.Uint())

	case reflect.Float32, reflect.Float64:
		return color.YellowString("%f", v.Float())

	case reflect.Bool:
		return color.MagentaString("%t", v.Bool())

	case reflect.Slice, reflect.Array:
		return formatSlice(v)

	case reflect.Map:
		return formatMap(v)

	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func formatStruct(v reflect.Value) string {
	t := v.Type()

	var parts []string

	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)
		fieldVal := v.Field(i)

		if !fieldVal.CanInterface() {
			continue
		}

		if fieldVal.IsZero() {
			continue
		}

		name := fieldType.Name

		if tag := fieldType.Tag.Get("json"); tag != "" {
			name = strings.Split(tag, ",")[0]
			if name == "-" {
				continue
			}
		}

		part := fieldColor.Sprint(name) + "=" + formatValue(fieldVal)
		parts = append(parts, part)
	}

	return typeColor.Sprint(t.Name()) + "{" + strings.Join(parts, " ") + "}"
}

func formatSlice(v reflect.Value) string {
	var items []string
	for i := 0; i < v.Len(); i++ {
		items = append(items, formatValue(v.Index(i)))
	}
	return color.BlueString("[%s]", strings.Join(items, ", "))
}

func formatMap(v reflect.Value) string {
	var items []string
	for _, key := range v.MapKeys() {
		k := formatValue(key)
		val := formatValue(v.MapIndex(key))
		items = append(items, k+":"+val)
	}
	return color.BlueString("{%s}", strings.Join(items, ", "))
}
