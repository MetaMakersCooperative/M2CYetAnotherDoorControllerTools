package windows

type Orientation struct {
	Top    int
	Left   int
	Bottom int
	Right  int
}

type Border struct {
	Top    bool
	Left   bool
	Bottom bool
	Right  bool
}

type Window struct {
	Height  int
	Width   int
	Padding Orientation
	Margin  Orientation
	Border  Border
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
