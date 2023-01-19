package epd

// Based on https://github.com/waveshare/e-Paper/blob/master/RaspberryPi_JetsonNano/c/lib/e-Paper/EPD_1in54_V2.c
// This code is quite messy and should be refactored if it is to be retained

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"time"

	"github.com/MaxHalford/halfgone"
	"github.com/disintegration/imaging"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/ssd1306/image1bit"
	"periph.io/x/host/v3"
)

// waveform full refresh
var wf_full_1in54 = []byte{
	0x80, 0x48, 0x40, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x40, 0x48, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x80, 0x48, 0x40, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x40, 0x48, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0xA, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x8, 0x1, 0x0, 0x8, 0x1, 0x0, 0x2,
	0xA, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x0, 0x0, 0x0,
	0x22, 0x17, 0x41, 0x0, 0x32, 0x20,
}

// waveform partial refresh(fast)
var wf_partial_1in54_0 = []byte{
	0x0, 0x40, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x80, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x40, 0x40, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0xF, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x0, 0x0, 0x0,
	0x02, 0x17, 0x41, 0xB0, 0x32, 0x28,
}

type epd154v2 struct {
	c    spi.Conn
	dc   gpio.PinOut
	cs   gpio.PinOut
	rst  gpio.PinOut
	busy gpio.PinIO

	image             *image1bit.VerticalLSB
	last_init_partial bool // what was the last init mode we used?
}

func NewEPD154V2(spi_bus string, dc, cs, rst gpio.PinOut, busy gpio.PinIO) (EPD, error) {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		return nil, err
	}

	b, err := spireg.Open(spi_bus)
	if err != nil {
		return nil, err
	}

	epd, err := NewEPD154V2FromSPI(b, dc, cs, rst, busy)
	if err != nil {
		b.Close()
		return nil, err
	}
	return epd, nil
}

func NewEPD154V2FromSPI(s spi.Port, dc, cs, rst gpio.PinOut, busy gpio.PinIO) (EPD, error) {
	if dc == gpio.INVALID {
		return nil, errors.New("epd: use nil for dc to use 3-wire mode, do not use gpio.INVALID")
	}

	c, err := s.Connect(20*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, err
	}

	if err := dc.Out(gpio.Low); err != nil {
		return nil, err
	}

	e := &epd154v2{
		c:                 c,
		dc:                dc,
		cs:                cs,
		rst:               rst,
		busy:              busy,
		last_init_partial: false,
		image:             image1bit.NewVerticalLSB(image.Rectangle{image.Point{0, 0}, image.Point{200, 200}}),
	}
	// TODO: track b & close it on EPD154.Close

	rst.Out(gpio.High)
	dc.Out(gpio.Low)
	cs.Out(gpio.High)
	busy.In(gpio.PullDown, gpio.NoEdge)

	e.init()
	e.clear()

	return e, nil
}

func (e *epd154v2) reset() {
	e.rst.Out(gpio.High)
	time.Sleep(20 * time.Millisecond)
	e.rst.Out(gpio.Low)
	time.Sleep(2 * time.Millisecond)
	e.rst.Out(gpio.High)
	time.Sleep(20 * time.Millisecond)
}

func (e *epd154v2) sendCommand(cmd byte) error {
	e.dc.Out(gpio.Low)
	e.cs.Out(gpio.Low)
	if err := e.c.Tx([]byte{cmd}, nil); err != nil {
		return err
	}
	e.cs.Out(gpio.High)
	return nil
}

/*
*****************************************************************************
function :	send data
parameter:

	Data : Write data

*****************************************************************************
*/
func (e *epd154v2) sendData(data byte) error {
	return e.sendDataBulk([]byte{data})
}

func (e *epd154v2) sendDataBulk(data []byte) error {
	e.dc.Out(gpio.High)
	e.cs.Out(gpio.Low)
	if err := e.c.Tx(data, nil); err != nil {
		return err
	}
	e.cs.Out(gpio.High)
	return nil
}

/*
*****************************************************************************
function :	Wait until the busy_pin goes LOW
parameter:
*****************************************************************************
*/
func (e *epd154v2) readBusy() {
	for e.busy.Read() == gpio.High {
		time.Sleep(time.Millisecond)
	}
}

/*
*****************************************************************************
function :	Turn On Display full
parameter:
*****************************************************************************
*/
func (e *epd154v2) turnOnDisplay() {
	e.sendCommand(0x22)
	e.sendData(0xc7)
	e.sendCommand(0x20)
	e.readBusy()
}

/*
*****************************************************************************
function :	Turn On Display part
parameter:
*****************************************************************************
*/
func (e *epd154v2) turnOnDisplayPart() {
	e.sendCommand(0x22)
	e.sendData(0xcF)
	e.sendCommand(0x20)
	e.readBusy()
}

func (e *epd154v2) lut(lut []byte) {
	e.sendCommand(0x32)
	e.sendDataBulk(lut[0:153])
	e.readBusy()
}

func (e *epd154v2) setLut(lut []byte) {
	e.lut(lut)

	e.sendCommand(0x3f)
	e.sendData(lut[153])

	e.sendCommand(0x03)
	e.sendData(lut[154])

	e.sendCommand(0x04)
	e.sendData(lut[155])
	e.sendData(lut[156])
	e.sendData(lut[157])

	e.sendCommand(0x2c)
	e.sendData(lut[158])
}

func (e *epd154v2) setWindows(xstart int, ystart int, xend int, yend int) {
	e.sendCommand(0x44) // SET_RAM_X_ADDRESS_START_END_POSITION
	e.sendData(byte(xstart >> 3))
	e.sendData(byte(xend >> 3))

	e.sendCommand(0x45) // SET_RAM_Y_ADDRESS_START_END_POSITION
	e.sendData(byte(ystart))
	e.sendData(byte(ystart >> 8))
	e.sendData(byte(yend))
	e.sendData(byte(yend >> 8))
}

func (e *epd154v2) setCursor(xstart int, ystart int) {
	e.sendCommand(0x4E) // SET_RAM_X_ADDRESS_COUNTER
	e.sendData(byte(xstart))

	e.sendCommand(0x4F) // SET_RAM_Y_ADDRESS_COUNTER
	e.sendData(byte(ystart))
	e.sendData(byte((ystart >> 8)))
}

/*
*****************************************************************************
function :	Initialize the e-Paper register
parameter:
*****************************************************************************
*/
func (e *epd154v2) init() {
	e.reset()

	e.readBusy()
	e.sendCommand(0x12) //SWRESET
	e.readBusy()

	e.sendCommand(0x01) //Driver output control
	e.sendData(0xC7)
	e.sendData(0x00)
	e.sendData(0x01)

	e.sendCommand(0x11) //data entry mode
	e.sendData(0x01)

	e.setWindows(0, e.Bounds().Dy(), e.Bounds().Dx(), 0)

	e.sendCommand(0x3C) //BorderWavefrom
	e.sendData(0x01)

	e.sendCommand(0x18)
	e.sendData(0x80)

	e.sendCommand(0x22) // //Load Temperature and waveform setting.
	e.sendData(0xB1)
	e.sendCommand(0x20)

	e.setCursor(0, e.Bounds().Dy()-1)
	e.readBusy()

	e.setLut(wf_full_1in54)
	e.last_init_partial = false
}

/*
*****************************************************************************
function :	Initialize the e-Paper register (Partial display)
parameter:
*****************************************************************************
*/
func (e *epd154v2) initPartial() {
	e.reset()
	e.readBusy()

	e.setLut(wf_partial_1in54_0)
	e.sendCommand(0x37)
	e.sendData(0x00)
	e.sendData(0x00)
	e.sendData(0x00)
	e.sendData(0x00)
	e.sendData(0x00)
	e.sendData(0x40)
	e.sendData(0x00)
	e.sendData(0x00)
	e.sendData(0x00)
	e.sendData(0x00)

	e.sendCommand(0x3C) //BorderWavefrom
	e.sendData(0x80)

	e.sendCommand(0x22)
	e.sendData(0xc0)
	e.sendCommand(0x20)
	e.readBusy()

	e.last_init_partial = true
}

/*
*****************************************************************************
function :	Clear screen
parameter:
*****************************************************************************
*/
func (e *epd154v2) clear() {
	e.sendCommand(0x24)
	e.sendImage(&image.Uniform{color.Black})

	e.sendCommand(0x26)
	e.sendImage(&image.Uniform{color.Black})

	e.turnOnDisplay()
}

/******************************************************************************
function :	Sends the image buffer in RAM to e-Paper and displays
parameter:
******************************************************************************/

func pixelisset(c color.Color) bool {
	r, g, b, a := c.RGBA()
	set := a >= 0x80 && (r > 0x20 || g > 0x20 || b > 0x20)
	return set
}

func (e *epd154v2) sendImage(img image.Image) {
	b := e.Bounds()
	tosend := make([]byte, b.Dx()*b.Dy()/8)
	for y := 0; y < b.Dy(); y++ {
		bytetosend := byte(0)
		for x := 0; x < b.Dx(); x++ {
			if pixelisset(img.At(x, y)) {
				bytetosend |= 0x80 >> (x % 8)
			}
			if x%8 == 7 {
				tosend[(y*b.Dx()+x)/8] = bytetosend
				bytetosend = 0
			}
		}
	}
	e.sendDataBulk(tosend)
}

func (e *epd154v2) display() {
	e.sendCommand(0x24)
	e.sendImage(e.image)
	e.turnOnDisplay()
}

/*
func (e *epd154v2) displayPartBaseImage(img image.Image) {
	e.sendCommand(0x24)
	e.sendImage(img)
	e.sendCommand(0x26)
	e.sendImage(img)
	e.turnOnDisplay()
}
*/

/*
*****************************************************************************
function :	Sends the image buffer in RAM to e-Paper and displays
parameter:
*****************************************************************************
*/
func (e *epd154v2) displayPart() {
	e.sendCommand(0x24)
	e.sendImage(e.image)
	e.turnOnDisplayPart()
}

/*
*****************************************************************************
function :	Enter sleep mode
parameter:
*****************************************************************************
*/
func (e *epd154v2) sleep() {
	e.sendCommand(0x10) //enter deep sleep
	e.sendData(0x01)
	time.Sleep(100 * time.Millisecond)
}

func (e *epd154v2) UpdateDisplay(img image.Image, partial bool) {
	// If we've changed mode, reinitialize
	if partial && !e.last_init_partial {
		e.initPartial()
	}
	if !partial && e.last_init_partial {
		e.init()
	}

	bounds := e.image.Bounds()
	gray := image.NewGray(bounds)
	if bounds != img.Bounds() {
		scaled := imaging.Fit(img, bounds.Max.X, bounds.Max.Y, imaging.Lanczos)
		draw.Draw(gray, bounds, scaled, image.Point{}, draw.Src)
	} else {
		draw.Draw(gray, bounds, img, image.Point{}, draw.Src)
	}

	// TODO: Should only dither if img isn't already B&W
	draw.Draw(e.image, bounds, halfgone.FloydSteinbergDitherer{}.Apply(gray), image.Point{}, draw.Src)

	if partial {
		e.displayPart()
	} else {
		e.display()
	}
}

func (e *epd154v2) Close() error {
	e.init()
	e.clear()
	e.sleep()
	return nil
}

func (e *epd154v2) Bounds() image.Rectangle {
	return e.image.Bounds()
}
