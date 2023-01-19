package epd

import (
	"image"
)

type EPD interface {
	UpdateDisplay(img image.Image, partial bool)
	Close() error
	Bounds() image.Rectangle
}
