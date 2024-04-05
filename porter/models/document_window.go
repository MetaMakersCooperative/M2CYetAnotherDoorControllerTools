package models

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DocumentWindow struct {
	logWindow    LogWindow
	statusWindow StatusWindow
	Window
}

func NewDocumentWindow(ctx context.Context, width int, height int) DocumentWindow {
	documentWindow := DocumentWindow{
		logWindow:    NewLogWindow(true),
		statusWindow: NewStatusWindow(ctx, false),
		Window: Window{
			focused: true,
			Width:   width,
			Height:  height,
			Margin:  Orientation{0, 0, 0, 0},
			Padding: Orientation{1, 2, 1, 2},
			Border:  Border{false, false, false, false},
		},
	}

	return documentWindow.UpdateDimensions(width, height)
}

func (documentWindow DocumentWindow) Update(msg tea.Msg) (DocumentWindow, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyRight {
			documentWindow.logWindow.Focus()
			documentWindow.statusWindow.Blur()
		} else if msg.Type == tea.KeyLeft {
			documentWindow.logWindow.Blur()
			documentWindow.statusWindow.Focus()
		}
	}

	var logWindowCmd tea.Cmd
	documentWindow.logWindow, logWindowCmd = documentWindow.logWindow.Update(msg)
	cmds = append(cmds, logWindowCmd)

	var statusWindowCmd tea.Cmd
	documentWindow.statusWindow, statusWindowCmd = documentWindow.statusWindow.Update(msg)
	cmds = append(cmds, statusWindowCmd)

	return documentWindow, tea.Batch(cmds...)
}

func (documentWindow DocumentWindow) UpdateDimensions(width int, height int) DocumentWindow {
	documentWindow.SetWidth(width)
	documentWindow.SetHeight(height)

	documentWindow.statusWindow = documentWindow.statusWindow.UpdateDimensions(
		35,
		documentWindow.GetInnerHeight(),
	)
	documentWindow.logWindow = documentWindow.logWindow.UpdateDimensions(
		documentWindow.GetInnerWidth()-35,
		documentWindow.GetInnerHeight(),
	)
	return documentWindow
}

func (documentWindow DocumentWindow) Render() string {
	if documentWindow.Height < 20 || documentWindow.Width < 80 {
		return lipgloss.NewStyle().
			Foreground(highlight).
			Render(
				"Terminal window needs to be larger than 80x20 to show information",
			)
	}

	docStyle := lipgloss.NewStyle().
		PaddingTop(documentWindow.Padding.Top).
		PaddingLeft(documentWindow.Padding.Left).
		PaddingBottom(documentWindow.Padding.Bottom).
		PaddingRight(documentWindow.Padding.Right)

	docStyle.MaxHeight(documentWindow.Height)
	docStyle.MaxWidth(documentWindow.Width)

	doc := strings.Builder{}

	doc.WriteString(lipgloss.JoinHorizontal(
		lipgloss.Top,
		documentWindow.statusWindow.Render(),
		documentWindow.logWindow.Render(),
	))

	doc.WriteString("\n\n")

	return docStyle.Render(doc.String())
}
