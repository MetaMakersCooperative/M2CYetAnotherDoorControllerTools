package models

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
)

type LogWindow struct {
	logs     []string
	err      error
	Viewport viewport.Model
	Window
}

func NewLogWindow() LogWindow {
	return LogWindow{
		logs:     make([]string, 0),
		Viewport: viewport.New(0, 0),
		Window: Window{
			Width:   0,
			Height:  0,
			Margin:  Orientation{1, 1, 0, 0},
			Padding: Orientation{1, 2, 1, 2},
			Border:  Border{true, true, true, true},
		},
	}
}

func (logsWindow *LogWindow) Log(prefix string, format string, args ...any) {
	logsWindow.logs = append(
		logsWindow.logs,
		fmt.Sprintf(
			"%s %s: "+format+"\n",
			append([]any{time.Now().Format("15:04:05"), prefix}, args...)...,
		),
	)
}

func (logsWindow *LogWindow) Info(format string, args ...any) {
	logsWindow.Log("INFO", format, args...)
}

func (logsWindow *LogWindow) Warn(format string, args ...any) {
	logsWindow.Log("WARN", format, args...)
}

func (logsWindow *LogWindow) Error(format string, args ...any) {
	logsWindow.Log("ERROR", format, args...)
}

func (logsWindow *LogWindow) Render() string {
	cursor := 0
	text := ""
	for cursor < len(logsWindow.logs) {
		text += wrap.String(
			wordwrap.String(logsWindow.logs[cursor], logsWindow.GetInnerWidth()),
			logsWindow.GetInnerWidth(),
		)
		cursor += 1
	}
	return text
}
