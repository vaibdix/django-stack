package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	mu     sync.Mutex
	level  Level
	output io.Writer
	json   bool
	color  bool
	prefix string
}

var defaultLogger *Logger
var once sync.Once

func Default() *Logger {
	once.Do(func() {
		defaultLogger = New(INFO)
	})
	return defaultLogger
}

func New(level Level) *Logger {
	return &Logger{
		level:  level,
		output: os.Stdout,
		json:   false,
		color:  true,
	}
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

func (l *Logger) SetJSON(json bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.json = json
}

func (l *Logger) SetColor(color bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.color = color
}

func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

func (l *Logger) log(level Level, msg string, args ...any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := level.String()

	if l.json {
		fmt.Fprintf(l.output, `{"time":"%s","level":"%s","message":"%s"}`+"\n",
			timestamp, levelStr, escapeJSON(msg))
	} else {
		var icon string
		var colorCode string
		resetCode := "\033[0m"

		switch level {
		case DEBUG:
			icon = "○"
			colorCode = "\033[36m"
		case INFO:
			icon = "●"
			colorCode = "\033[32m"
		case WARN:
			icon = "◐"
			colorCode = "\033[33m"
		case ERROR:
			icon = "✗"
			colorCode = "\033[31m"
		}

		if l.color {
			fmt.Fprintf(l.output, "%s[%s] %s%s%s %s%s%s %s\n",
				"\033[90m", timestamp, resetCode,
				colorCode, icon, resetCode,
				levelStr, resetCode,
				l.prefix+msg)
		} else {
			fmt.Fprintf(l.output, "[%s] %s %s %s\n",
				timestamp, levelStr, l.prefix, msg)
		}
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	l.log(DEBUG, msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.log(INFO, msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.log(WARN, msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.log(ERROR, msg, args...)
}

func (l *Logger) Success(msg string, args ...any) {
	if INFO < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	if l.color {
		fmt.Fprintf(l.output, "  \033[32m✓\033[0m %s\n", msg)
	} else {
		fmt.Fprintf(l.output, "  [OK] %s\n", msg)
	}
}

func (l *Logger) Skip(msg string, args ...any) {
	if INFO < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	if l.color {
		fmt.Fprintf(l.output, "  \033[90m○\033[0m %s\n", msg)
	} else {
		fmt.Fprintf(l.output, "  [SKIP] %s\n", msg)
	}
}

func (l *Logger) DryRun(msg string, args ...any) {
	if INFO < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	if l.color {
		fmt.Fprintf(l.output, "  \033[36m[DRY RUN]\033[0m %s\n", msg)
	} else {
		fmt.Fprintf(l.output, "  [DRY RUN] %s\n", msg)
	}
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

var (
	Debug = Default().Debug
	Info  = Default().Info
	Warn  = Default().Warn
	Error = Default().Error
)

func SetLevel(level Level) {
	Default().SetLevel(level)
}

func SetJSON(json bool) {
	Default().SetJSON(json)
}

func Success(msg string, args ...any) {
	Default().Success(msg, args...)
}

func Skip(msg string, args ...any) {
	Default().Skip(msg, args...)
}

func DryRun(msg string, args ...any) {
	Default().DryRun(msg, args...)
}
