package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"strings"
	"time"

	"image/color"
	"image/gif"
	_ "image/png"

	"github.com/AndreRenaud/ftdi_eink/epd"

	"github.com/disintegration/imaging"
	"periph.io/x/conn/v3/gpio"
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
func getGifFromFilePath(filePath string) (*gif.GIF, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := gif.DecodeAll(f)
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

	ft232h, ok := all[0].(*ftdi.FT232H)
	if !ok {
		log.Fatalf("no FT232H device on the USB bus (available: %v)", all)
	}
	defer ft232h.Halt()

	s, err := ft232h.SPI()
	if err != nil {
		log.Fatalf("spi: %s", err)
	}
	defer s.Close()

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
		log.Fatalf("rst: %s", err)
	}
	busy, err := findGPIO(ft232h, "FT232H.C3")
	if err != nil {
		log.Fatalf("busy: %s", err)
	}

	var disp epd.EPD

	disp, err = epd.NewEPD154V2FromSPI(s, dc, cs, rst, busy)
	if err != nil {
		log.Fatalf("NewEPD: %s", err)
	}

	if strings.HasSuffix(*image_filename, ".gif") {
		log.Printf("Detected GIF - assuming an infinite loop")
		img, err := getGifFromFilePath(*image_filename)
		if err != nil {
			log.Fatalf("load gif: %s", err)
		}
		first := true
		for {
			for i := 0; i < len(img.Image); i++ {
				delay := time.Duration(img.Delay[i]*10) * time.Millisecond
				next := time.Now().Add(delay)
				log.Printf("Drawing frame %d of %d (%s delay)", i, len(img.Image), delay)
				frame := imaging.Rotate(img.Image[i], float64(*rotate%360), color.Transparent)
				disp.UpdateDisplay(frame, !first)
				time.Sleep(time.Until(next))
				first = false
			}
		}
	} else {

		img, err := getImageFromFilePath(*image_filename)
		if err != nil {
			log.Fatalf("load image: %s", err)
		}
		if *rotate != 0 {
			img = imaging.Rotate(img, float64(*rotate%360), color.Transparent)
		}
		disp.UpdateDisplay(img, false)

		if *spin {
			for i := 0.0; i <= 360; i += 5 {
				rot := imaging.Rotate(img, i, color.Transparent)
				disp.UpdateDisplay(rot, true)
			}
		}
	}

	// We deliberately don't close it, as that will clear the screen
	//disp.Close()
}
