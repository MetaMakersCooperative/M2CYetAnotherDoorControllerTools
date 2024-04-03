package models

type KeyLabelPair struct {
	Key   string
	Label string
}

type Options struct {
	options map[string]Checkbox
	order   []string
	active  int
}

func NewOptions(pairs ...KeyLabelPair) Options {
	options := make(map[string]Checkbox, 0)
	order := make([]string, 0)
	for index, pair := range pairs {
		options[pair.Key] = Checkbox{Label: pair.Label}
		order = append(order, pair.Key)
		if index == 0 {
			options[pair.Key] = options[pair.Key].ToggleActive()
		}
	}

	return Options{
		options: options,
		order:   order,
		active:  0,
	}
}

func (options Options) Render() string {
	text := ""
	for _, key := range options.order {
		text += options.options[key].Render() + "\n"
	}
	return text
}
