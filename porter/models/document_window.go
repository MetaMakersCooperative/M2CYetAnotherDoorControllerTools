package models

type DocumentWindow struct {
	Window
}

func NewDocumentWindow() DocumentWindow {
	return DocumentWindow{
		Window: Window{
			Width:   0,
			Height:  0,
			Margin:  Orientation{0, 0, 0, 0},
			Padding: Orientation{1, 2, 1, 2},
			Border:  Border{false, false, false, false},
		},
	}
}
