package models

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyLabelPair struct {
	Key   string
	Label string
}

type Options struct {
	focused     bool
	options     map[string]Checkbox
	order       []string
	active      int
	lastToggled int
	isRadio     bool
	keyBindings
}

type keyBindings struct {
	up    key.Binding
	down  key.Binding
	check key.Binding
}

func NewOptions(isRadio bool, pairs ...KeyLabelPair) Options {
	options := make(map[string]Checkbox, 0)
	order := make([]string, 0)
	for index, pair := range pairs {
		options[pair.Key] = Checkbox{Label: pair.Label, IsRadio: isRadio}
		order = append(order, pair.Key)
		if isRadio && index == 0 {
			options[pair.Key] = options[pair.Key].Toggle()
		}
	}

	return Options{
		options:     options,
		order:       order,
		active:      0,
		lastToggled: 0,
		focused:     false,
		isRadio:     isRadio,
		keyBindings: keyBindings{
			up:    key.NewBinding(key.WithKeys("k", "up")),
			down:  key.NewBinding(key.WithKeys("j", "down")),
			check: key.NewBinding(key.WithKeys(" ", "enter")),
		},
	}
}

func (options Options) Render() string {
	text := ""
	for index, key := range options.order {
		text += options.options[key].Render()
		if index < len(options.order)-1 {
			text += "\n"
		}
	}
	return optionsStyle.Render(text)
}

func (options Options) toggleFocusAt(position int) Options {
	if len(options.options) == 0 {
		return options
	}
	options.options[options.order[position]] = options.options[options.order[position]].ToggleFocus()
	options.active = position
	return options
}

func (options Options) Blur() Options {
	options.focused = false
	options.options[options.order[options.active]] = options.options[options.order[options.active]].Blur()
	return options
}

func (options Options) Focus() Options {
	options.focused = true
	options.options[options.order[options.active]] = options.options[options.order[options.active]].Focus()
	return options
}

func (options Options) ToggleFocus() Options {
	options.focused = !options.focused
	options.options[options.order[options.active]] = options.options[options.order[options.active]].ToggleFocus()
	return options
}

func (options Options) Update(msg tea.Msg) Options {
	if !options.focused || len(options.options) == 0 {
		options.options[options.order[options.active]] = options.options[options.order[options.active]].Blur()
		return options
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, options.keyBindings.up):
			if options.active-1 < 0 {
				break
			}
			options = options.toggleFocusAt(options.active).
				toggleFocusAt(options.active - 1)
		case key.Matches(msg, options.keyBindings.down):
			if options.active+1 >= len(options.options) {
				break
			}
			options = options.toggleFocusAt(options.active).
				toggleFocusAt(options.active + 1)
		case key.Matches(msg, options.keyBindings.check):
			if options.isRadio {
				if options.lastToggled == options.active {
					break
				}
				options.options[options.order[options.lastToggled]] = options.options[options.order[options.lastToggled]].Toggle()
				options.lastToggled = options.active
			}
			options.options[options.order[options.active]] = options.options[options.order[options.active]].Toggle()
		}
	}
	return options
}
