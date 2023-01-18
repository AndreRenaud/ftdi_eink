# FTDI EInk

This is a minimal tool for connecting an FT232H to an EPD154 EInk display. This makes it easy to drive the EInk from a PC via USB (the FT232H provides the SPI & GPIOs necessary).

# Wiring

Using a UM232H (rev 1.0) - note: VIO & 3v3 should be shorted together on the UM232H, and 5V & USB should also be shorted together. This is necessary to power the system on.


| UM232H Pin  | GooDisplay DESPI-C02 Pin | Description     |
| ----------- | ------------------------ | --------------- |
| 3V3         | 3.3V                     | Power           |
| GND         | GND                      | Ground          |
| AD0         | SCK                      | SPI Clock       |
| AD1         | SDI                      | SPI MOSI        |
| AC0         | D/C                      | D/C GPIO        |
| AC1         | CS                       | Chipselect GPIO |
| AC2         | RES                      | Reset GPIO      |
| AC3         | BUSY                     | Busy GPIO       |

