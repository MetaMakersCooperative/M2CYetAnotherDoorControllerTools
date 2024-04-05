package models

import "github.com/charmbracelet/lipgloss"

type Orientation struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

type Border struct {
	Top    bool
	Right  bool
	Bottom bool
	Left   bool
}

type Window struct {
	Height  int
	Width   int
	Padding Orientation
	Margin  Orientation
	Border  Border
	focused bool
}

func (window *Window) IsFocused() bool {
	return window.focused
}

func (window *Window) ToggleFocus() {
	window.focused = !window.focused
}

func (window *Window) Focus() {
	window.focused = true
}

func (window *Window) Blur() {
	window.focused = false
}

func (window *Window) SetHeight(height int) {
	window.Height = height - window.Margin.Top - window.Margin.Bottom
	if window.Border.Top {
		window.Height -= 1
	}
	if window.Border.Bottom {
		window.Height -= 1
	}
}

func (window *Window) SetWidth(width int) {
	window.Width = width - window.Margin.Left - window.Margin.Right
	if window.Border.Left {
		window.Width -= 1
	}
	if window.Border.Right {
		window.Width -= 1
	}
}

func (window *Window) GetInnerWidth() int {
	return window.Width - window.Padding.Left - window.Padding.Right
}

func (window *Window) GetInnerHeight() int {
	return window.Height - window.Padding.Top - window.Padding.Bottom
}

func (window *Window) Render(content ...string) string {
	style := windowStyle.
		Copy().
		Height(window.Height).
		Width(window.Width).
		MarginTop(window.Margin.Top).
		MarginLeft(window.Margin.Left).
		MarginBottom(window.Margin.Bottom).
		MarginRight(window.Margin.Right).
		PaddingTop(window.Padding.Top).
		PaddingLeft(window.Padding.Left).
		PaddingBottom(window.Padding.Bottom).
		PaddingRight(window.Padding.Right).
		Border(lipgloss.RoundedBorder()).
		BorderTop(window.Border.Top).
		BorderRight(window.Border.Right).
		BorderBottom(window.Border.Bottom).
		BorderLeft(window.Border.Left)

	if window.IsFocused() {
		style = style.BorderForeground(lipgloss.Color("#F25D94"))
	}

	return style.Render(content...)
}
