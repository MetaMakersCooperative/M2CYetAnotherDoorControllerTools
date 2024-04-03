package models

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type Checkbox struct {
	checked bool
	active  bool
	Label   string
}

func (checkBox Checkbox) Render() string {
	var style lipgloss.Style
	if checkBox.active {
		style = checkboxHighlightStyle
	} else {
		style = checkboxStyle
	}

	var check string
	if checkBox.checked {
		check = "x"
	} else {
		check = " "
	}

	return style.Render(fmt.Sprintf("[%s] %s", check, checkBox.Label))
}

func (checkbox Checkbox) IsChecked() bool {
	return checkbox.checked
}

func (checkbox Checkbox) Toggle() Checkbox {
	checkbox.checked = !checkbox.checked
	return checkbox
}

func (checkbox Checkbox) ToggleActive() Checkbox {
	checkbox.active = !checkbox.active
	return checkbox
}
