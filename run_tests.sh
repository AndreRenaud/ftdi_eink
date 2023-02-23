#!/bin/bash

while : ; do
	echo "Press enter to draw white image"
	read
	go run ./ftdi_demo.go -image white
	sleep 1 # So FTDI has time to rescan
	echo "Press enter to draw black image"
	read
	go run ./ftdi_demo.go -image black
	sleep 1 # So FTDI has time to rescan
	echo "Press enter to draw test image"
	read
	go run ./ftdi_demo.go -image output.png
	sleep 1 # So FTDI has time to rescan
	echo "Press button and confirm LED"
	read

	echo "Unit complete - move on to next one"
done

