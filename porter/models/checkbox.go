package models

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type Checkbox struct {
	checked bool
	focused bool
	Label   string
	IsRadio bool
}

func (checkBox Checkbox) Render() string {
	var style lipgloss.Style
	if checkBox.focused {
		style = checkboxHighlightStyle
	} else {
		style = checkboxStyle
	}

	var check string
	if checkBox.checked {
		if checkBox.IsRadio {
			check = "*"
		} else {
			check = "x"
		}
	} else {
		check = " "
	}

	if checkBox.IsRadio {
		return style.Render(fmt.Sprintf("(%s) %s", check, checkBox.Label))
	} else {
		return style.Render(fmt.Sprintf("[%s] %s", check, checkBox.Label))
	}
}

func (checkbox Checkbox) IsChecked() bool {
	return checkbox.checked
}

func (checkbox Checkbox) Toggle() Checkbox {
	checkbox.checked = !checkbox.checked
	return checkbox
}

func (checkbox Checkbox) ToggleFocus() Checkbox {
	checkbox.focused = !checkbox.focused
	return checkbox
}

func (checkbox Checkbox) Focus() Checkbox {
	checkbox.focused = true
	return checkbox
}

func (checkbox Checkbox) Blur() Checkbox {
	checkbox.focused = false
	return checkbox
}
