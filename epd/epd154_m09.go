package epd

// Based on https://github.com/GoodDisplay/E-paper-Display-Library-of-GoodDisplay/blob/main/Monochrome_E-paper-Display/1.54inch_JD79653_GDEW0154M09_200x200/Arduino/GDEW0154M09_Arduino.ino

// TODO: Try using this instead? https://github.com/vamoosebbf/sp_eink/blob/master/src/epd/epd.c

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"log"
	"time"

	"github.com/MaxHalford/halfgone"
	"github.com/disintegration/imaging"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/devices/v3/ssd1306/image1bit"
)

type epd154m09 struct {
	c    spi.Conn
	dc   gpio.PinOut
	cs   gpio.PinOut
	rst  gpio.PinOut
	busy gpio.PinIO

	image             *image1bit.VerticalLSB
	last_init_partial bool // what was the last init mode we used?
}

func NewEPD154M09FromSPI(s spi.Port, dc, cs, rst gpio.PinOut, busy gpio.PinIO) (EPD, error) {
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

	e := &epd154m09{
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

	log.Printf("about to init")
	e.init()
	log.Printf("about to clear")
	e.clear()

	log.Printf("created screen")
	return e, nil
}

func (e *epd154m09) clear() {
	e.sendCommand(0x10)
	e.sendImage(&image.Uniform{color.Black})
	e.sendCommand(0x13)
	e.sendImage(&image.Uniform{color.Black})
	e.sendCommand(0x12) //DISPLAY REFRESH
	time.Sleep(10 * time.Millisecond)
	//delay(10)           //!!!The delay here is necessary, 200uS at least!!!
	e.readBusy()

}

//Tips//
/*
1.When the e-paper is refreshed in full screen, the picture flicker is a normal phenomenon, and the main function is to clear the display afterimage in the previous picture.
2.When the partial refresh is performed, the screen does not flash.
3.After the e-paper is refreshed, you need to put it into sleep mode, please do not delete the sleep command.
4.Please do not take out the electronic paper when power is on.
5.Wake up from sleep, need to re-initialize the e-paper.
6.When you need to transplant the driver, you only need to change the corresponding IO. The BUSY pin is the input mode and the others are the output mode.
*/
/*
void loop() {

  while(1)
  {
      //Clear
      EPD_init(); //EPD init
      PIC_display_Clean();//EPD Clear
      EPD_sleep();//EPD_sleep,Sleep instruction is necessary, please do not delete!!!
      delay(2000); //2s

      EPD_init(); //EPD init
      PIC_display(gImage_1,gImage_1,0);//EPD_picture1
      EPD_sleep();//EPD_sleep,Sleep instruction is necessary, please do not delete!!!
      delay(2000); //2s

      EPD_init(); //EPD init
      PIC_display(gImage_1,gImage_2,1);//EPD_picture1
      EPD_sleep();//EPD_sleep,Sleep instruction is necessary, please do not delete!!!
      delay(2000); //2s
      //Clear
      EPD_init(); //EPD init
      PIC_display_Clean();//EPD Clear
      EPD_sleep();//EPD_sleep,Sleep instruction is necessary, please do not delete!!!
      delay(2000); //2s
      while(1);
  }


}
*/

//////////////////////SPI///////////////////////////////////

func (e *epd154m09) sendCommand(cmd byte) error {
	e.dc.Out(gpio.Low)
	e.cs.Out(gpio.Low)
	if err := e.c.Tx([]byte{cmd}, nil); err != nil {
		return err
	}
	e.cs.Out(gpio.High)
	return nil
}

func (e *epd154m09) sendData(data byte) error {
	return e.sendDataBulk([]byte{data})
}

func (e *epd154m09) sendDataBulk(data []byte) error {
	e.dc.Out(gpio.High)
	e.cs.Out(gpio.Low)
	if err := e.c.Tx(data, nil); err != nil {
		return err
	}
	e.cs.Out(gpio.High)
	return nil
}

// ///////////////EPD settings Functions/////////////////////
func (e *epd154m09) reset() {
	e.rst.Out(gpio.Low)
	time.Sleep(10 * time.Millisecond)
	e.rst.Out(gpio.High)
	time.Sleep(10 * time.Millisecond)
}

func (e *epd154m09) init() {
	e.reset()
	time.Sleep(100 * time.Millisecond)

	e.sendCommand(0x00) // panel setting
	e.sendData(0xDf)
	e.sendData(0x0e)

	e.sendCommand(0x4D) //FITIinternal code
	e.sendData(0x55)

	e.sendCommand(0xaa)
	e.sendData(0x0f)

	e.sendCommand(0xE9)
	e.sendData(0x02)

	e.sendCommand(0xb6)
	e.sendData(0x11)

	e.sendCommand(0xF3)
	e.sendData(0x0a)

	e.sendCommand(0x61) //resolution setting
	e.sendData(0xc8)
	e.sendData(0x00)
	e.sendData(0xc8)

	e.sendCommand(0x60) //Tcon setting
	e.sendData(0x00)

	e.sendCommand(0x50)
	e.sendData(0x97) //

	e.sendCommand(0xE3)
	e.sendData(0x00)

	e.sendCommand(0x04) //Power on
	time.Sleep(100 * time.Millisecond)
	e.readBusy()

}
func (e *epd154m09) refresh() {
	e.sendCommand(0x12) //DISPLAY REFRESH
	time.Sleep(10 * time.Millisecond)
	e.readBusy()
}

func (e *epd154m09) sleep() {
	e.sendCommand(0x02) //power off
	e.readBusy()
	//Part2 Increase the time delay
	time.Sleep(1000 * time.Millisecond)
	e.sendCommand(0x07) //deep sleep
	e.sendData(0xA5)
}

func (e *epd154m09) Close() error {
	return nil
}

func (e *epd154m09) Bounds() image.Rectangle {
	return e.image.Bounds()
}

func (e *epd154m09) sendImage(img image.Image) {
	b := e.Bounds()
	for y := 0; y < b.Dy(); y++ {
		bytetosend := byte(0)
		for x := 0; x < b.Dx(); x++ {
			if pixelisset(img.At(x, y)) {
				bytetosend |= 0x80 >> (x % 8)
			}
			if x%8 == 7 {
				e.sendData(bytetosend)
				bytetosend = 0
			}
		}
	}
}

func (e *epd154m09) UpdateDisplay(img image.Image, partial bool) {
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

	//if(mode==0)  //mode0:Refresh picture1
	//{
	e.sendCommand(0x10)
	e.sendImage(&image.Uniform{color.White})
	e.sendCommand(0x13)
	e.sendImage(e.image)
	//}
	/*
	    else if(mode==1)  //mode0:Refresh picture2...
	    {
	      e.sendCommand(0x10);
	      for(i=0;i<5000;i++)
	      {
	          e.sendData(pgm_read_byte(&old_data[i]));
	      }
	      e.sendCommand(0x13);
	      for(i=0;i<5000;i++)
	      {
	          e.sendData(pgm_read_byte(&new_data[i]));
	      }
	    }

	   else if(mode==2)
	    {
	      e.sendCommand(0x10);
	      for(i=0;i<5000;i++)
	      {
	          e.sendData(pgm_read_byte(&old_data[i]));
	      }
	      e.sendCommand(0x13);
	      for(i=0;i<5000;i++)
	      {
	          e.sendData(0xff);
	      }
	    }
	   else if(mode==3)
	    {
	      e.sendCommand(0x10);
	      for(i=0;i<5000;i++)
	      {
	          e.sendData(0xff);
	      }
	      e.sendCommand(0x13);
	      for(i=0;i<5000;i++)
	      {
	          e.sendData(0xff);
	      }
	    }
	*/

	e.sendCommand(0x12) //DISPLAY REFRESH
	time.Sleep(10 * time.Millisecond)
	//delay(10)           //!!!The delay here is necessary, 200uS at least!!!
	e.readBusy()
}

/*
void PIC_display_Clean(void)

	{
	    unsigned int i,j;
	    for(j=0;j<2;j++)
	    {
	      e.sendCommand(0x10);
	      for(i=0;i<5000;i++)
	      {
	          e.sendData(0x00);
	      }
	      e.sendCommand(0x13);
	      for(i=0;i<5000;i++)
	      {
	          e.sendData(0xff);
	      }
	      e.sendCommand(0x12);     //DISPLAY REFRESH
	      delay(10);     //!!!The delay here is necessary, 200uS at least!!!
	      e.readBusy()
	    }

}
*/
func (e *epd154m09) readBusy() {
	log.Printf("Readbusy")
	for e.busy.Read() == gpio.High {
		log.Printf("still waiting")
		time.Sleep(time.Millisecond)
	}
	log.Printf("done busy")
}
