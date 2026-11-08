//go:build tinygo && stm32f4disco

package main

import "unsafe"

const ltdc = uintptr(0x40016800)
const layer = ltdc + 0x84

var framebuffer [320][240]uint16
var lcdOK bool

func gpio(n uintptr) uintptr { return 0x40020000 + n*0x400 }
func pinMode(port, pin, mode, af uint32) {
	base := gpio(uintptr(port))
	shift := pin * 2
	writeRegister(base, (readRegister(base)&^(3<<shift))|(mode<<shift))
	setRegisterBits(base+8, 3<<shift)
	clearRegisterBits(base+0xc, 3<<shift)
	if mode == 2 {
		a := base + 0x20 + uintptr(pin/8)*4
		s := (pin % 8) * 4
		writeRegister(a, (readRegister(a)&^(15<<s))|(af<<s))
	}
}
func sendByte(v byte, data bool) {
	if !lcdOK {
		return
	}
	shift := uint32(29)
	if data {
		shift = 13
	}
	writeRegister(gpio(3)+0x18, 1<<shift)
	writeRegister(gpio(2)+0x18, 1<<18)
	if !waitForRegister(spi5+8, 2, true) {
		lcdOK = false
		writeRegister(gpio(2)+0x18, 1<<2)
		return
	}
	writeRegister8(spi5+0xc, v)
	if !waitForRegister(spi5+8, 2, true) || !waitForRegister(spi5+8, 1<<7, false) {
		lcdOK = false
	}
	writeRegister(gpio(2)+0x18, 1<<2)
}
func command(cmd byte, data ...byte) {
	sendByte(cmd, false)
	for _, v := range data {
		sendByte(v, true)
	}
}
func lcdInit() bool {
	lcdOK = true
	setRegisterBits(rcc+0x30, 0x6f)
	setRegisterBits(rcc+0x44, (1<<20)|(1<<26))
	_ = readRegister(rcc + 0x44)
	pinMode(2, 2, 1, 0)
	pinMode(3, 13, 1, 0)
	writeRegister(gpio(2)+0x18, 1<<2)
	pinMode(5, 7, 2, 5)
	pinMode(5, 9, 2, 5)
	writeRegister(spi5, (1<<2)|(1<<8)|(1<<9)|(1<<3)|(1<<6))
	delay(50)
	command(0xca, 0xc3, 0x08, 0x50)
	command(0xcf, 0, 0xc1, 0x30)
	command(0xed, 0x64, 3, 0x12, 0x81)
	command(0xe8, 0x85, 0, 0x78)
	command(0xcb, 0x39, 0x2c, 0, 0x34, 2)
	command(0xf7, 0x20)
	command(0xea, 0, 0)
	command(0xb1, 0, 0x1b)
	command(0xb6, 0x0a, 0xa2)
	command(0xc0, 0x10)
	command(0xc1, 0x10)
	command(0xc5, 0x45, 0x15)
	command(0xc7, 0x90)
	command(0x36, 0xc8)
	command(0xf2, 0)
	command(0xb0, 0xc2)
	command(0xb6, 0x0a, 0xa7, 0x27, 4)
	command(0x2a, 0, 0, 0, 0xef)
	command(0x2b, 0, 0, 1, 0x3f)
	command(0xf6, 1, 0, 6)
	command(0x26, 1)
	command(0xe0, 0x0f, 0x29, 0x24, 0x0c, 0x0e, 0x09, 0x4e, 0x78, 0x3c, 0x09, 0x13, 0x05, 0x17, 0x11, 0)
	command(0xe1, 0, 0x16, 0x1b, 4, 0x11, 7, 0x31, 0x33, 0x42, 5, 0x0c, 0x0a, 0x28, 0x2f, 0x0f)
	command(0x11)
	delay(200)
	command(0x29)
	if !lcdOK {
		return false
	}
	pins := [...][3]uint32{{0, 3, 14}, {0, 4, 14}, {0, 6, 14}, {0, 11, 14}, {0, 12, 14}, {1, 0, 9}, {1, 1, 9}, {1, 8, 14}, {1, 9, 14}, {1, 10, 14}, {1, 11, 14}, {2, 6, 14}, {2, 7, 14}, {2, 10, 14}, {3, 3, 14}, {3, 6, 14}, {5, 10, 14}, {6, 6, 14}, {6, 7, 14}, {6, 10, 9}, {6, 11, 14}, {6, 12, 9}}
	for _, p := range pins {
		pinMode(p[0], p[1], 2, p[2])
	}
	writeRegister(rcc+4, (readRegister(rcc+4)&^63)|16)
	writeRegister(rcc+0x88, (192<<6)|(7<<24)|(4<<28))
	writeRegister(rcc+0x8c, (readRegister(rcc+0x8c)&^(3<<16))|(2<<16))
	setRegisterBits(rcc, 1<<28)
	if !waitForRegister(rcc, 1<<29, true) {
		return false
	}
	writeRegister(ltdc+8, (9<<16)|1)
	writeRegister(ltdc+0xc, (29<<16)|3)
	writeRegister(ltdc+0x10, (269<<16)|323)
	writeRegister(ltdc+0x14, (279<<16)|327)
	writeRegister(layer+4, (269<<16)|30)
	writeRegister(layer+8, (323<<16)|4)
	writeRegister(layer+0x10, 2)
	writeRegister(layer+0x14, 255)
	writeRegister(layer+0x1c, (4<<8)|5)
	writeRegister(layer+0x28, uint32(uintptr(unsafe.Pointer(&framebuffer[0][0]))))
	writeRegister(layer+0x2c, (480<<16)|483)
	writeRegister(layer+0x30, 320)
	writeRegister(layer, 1)
	writeRegister(ltdc+0x24, 1)
	setRegisterBits(ltdc+0x18, 1)
	return true
}
func lcdRect(x, y, w, h int, color uint16) {
	if x < 0 || y < 0 || x >= 320 || y >= 240 {
		return
	}
	if w > 320-x {
		w = 320 - x
	}
	if h > 240-y {
		h = 240 - y
	}
	for col := x; col < x+w; col++ {
		for row := y; row < y+h; row++ {
			framebuffer[319-col][row] = color
		}
	}
}
