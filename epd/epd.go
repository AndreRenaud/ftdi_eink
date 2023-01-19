package epd

import (
	"fmt"
	"image"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/spi"
)

type EPD interface {
	UpdateDisplay(img image.Image, partial bool)
	Close() error
	Bounds() image.Rectangle
}

var epd_types = map[string]func(spi.Port, gpio.PinOut, gpio.PinOut, gpio.PinOut, gpio.PinIO) (EPD, error){
	"154_v2":  NewEPD154V2FromSPI,
	"154_m09": NewEPD154M09FromSPI,
}

func SupportedTypes() []string {
	retval := make([]string, 0, len(epd_types))
	for k := range epd_types {
		retval = append(retval, k)
	}
	return retval
}

func NewEPDFromSPI(epd_type string, s spi.Port, dc, cs, rst gpio.PinOut, busy gpio.PinIO) (EPD, error) {
	for k, v := range epd_types {
		if k == epd_type {
			return v(s, dc, cs, rst, busy)
		}
	}
	return nil, fmt.Errorf("unknown epd type %q", epd_type)
}
