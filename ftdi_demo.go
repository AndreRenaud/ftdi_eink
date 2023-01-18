package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"

	"image/color"
	"image/draw"
	_ "image/png"

	"github.com/disintegration/imaging"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/devices/v3/ssd1306/image1bit"
	"periph.io/x/host/v3"
	"periph.io/x/host/v3/ftdi"
)

func findGPIO(ft232h *ftdi.FT232H, name string) (gpio.PinIO, error) {
	headers := ft232h.Header()
	for _, h := range headers {
		if h.Name() == name {
			return h, nil
		}
	}
	return nil, fmt.Errorf("no such gpio %s", name)
}

func getImageFromFilePath(filePath string) (image.Image, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	image, _, err := image.Decode(f)
	return image, err
}

func main() {
	image_filename := flag.String("image", "", "Image to draw on the EInk")
	rotate := flag.Int("rotate", 0, "Rotation angle")
	flag.Parse()

	if *image_filename == "" {
		log.Fatal("must supply --image")
	}

	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	all := ftdi.All()
	if len(all) == 0 {
		log.Fatal("found no FTDI device on the USB bus")
	}

	// Use channel A.
	ft232h, ok := all[0].(*ftdi.FT232H)
	if !ok {
		log.Fatal("not FTDI device on the USB bus")
	}

	s, err := ft232h.SPI()
	if err != nil {
		log.Fatalf("spi: %s", err)
	}

	c, err := s.Connect(5*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		log.Fatalf("Connect: %s", err)
	}

	dc, err := findGPIO(ft232h, "FT232H.C0")
	if err != nil {
		log.Fatalf("DC: %s", err)
	}
	cs, err := findGPIO(ft232h, "FT232H.C1")
	if err != nil {
		log.Fatalf("cs: %s", err)
	}
	rst, err := findGPIO(ft232h, "FT232H.C2")
	if err != nil {
		log.Fatalf("cs: %s", err)
	}
	busy, err := findGPIO(ft232h, "FT232H.C3")
	if err != nil {
		log.Fatalf("cs: %s", err)
	}

	epd, err := NewEPD154FromConn(c, dc, cs, rst, busy)
	if err != nil {
		log.Fatalf("NewEPD: %s", err)
	}

	img := image1bit.NewVerticalLSB(image.Rectangle{image.Point{0, 0}, image.Point{200, 200}})
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	//img := image.NewNRGBA(image.Rect(0, 0, 200, 200))
	// fill it white
	//draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	logo, err := getImageFromFilePath(*image_filename)
	if err != nil {
		log.Fatalf("load image: %s", err)
	}
	new := imaging.Rotate(logo, float64((*rotate+180)%360), color.Transparent)
	draw.Draw(img, img.Bounds(), new, image.Point{}, draw.Src)
	//draw.Draw(img, img.Bounds(), logo, image.Point{}, draw.Src)
	epd.UpdateDisplay(img, false)
	//epd.display(img)

	//epd.Close()
}
