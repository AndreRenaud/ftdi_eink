package main

import (
	"fmt"
	"log"

	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/host/v3"
	"periph.io/x/host/v3/ftdi"
)

func main() {
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
		log.Fatal(err)
	}

	c, err := s.Connect(physic.KiloHertz*100, spi.Mode3, 8)
	write := []byte{0x10, 0x00}
	read := make([]byte, len(write))
	if err := c.Tx(write, read); err != nil {
		log.Fatal(err)
	}
	// Use read.
	fmt.Printf("read: %v\n", read)

	cb, err := ft232h.CBusRead()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("cbus: 0x%x", cb)
}
