package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"

	"image/color"
	_ "image/png"

	"github.com/disintegration/imaging"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
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
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func main() {
	image_filename := flag.String("image", "", "Image to draw on the EInk")
	rotate := flag.Int("rotate", 0, "Rotation angle")
	spin := flag.Bool("spin", false, "If set, do a partial refresh 360 degree spin after the initial draw")

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

	img, err := getImageFromFilePath(*image_filename)
	if err != nil {
		log.Fatalf("load image: %s", err)
	}
	if *rotate != 0 {
		img = imaging.Rotate(img, float64(*rotate%360), color.Transparent)
	}
	epd.UpdateDisplay(img, false)

	if *spin {
		for i := 0.0; i <= 360; i += 5 {
			rot := imaging.Rotate(img, i, color.Transparent)
			epd.UpdateDisplay(rot, true)
		}

	}

	//epd.Close()
}
