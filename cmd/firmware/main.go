//go:build tinygo && stm32f4disco

package main

import (
	game "gyrobreakout"
	"runtime/volatile"
	"unsafe"
)

const (
	rcc  = uintptr(0x40023800)
	spi5 = uintptr(0x40015000)
	pc   = uintptr(0x40020800)
	pf   = uintptr(0x40021400)
)

func readRegister(a uintptr) uint32         { return volatile.LoadUint32((*uint32)(unsafe.Pointer(a))) }
func writeRegister(a uintptr, v uint32)     { volatile.StoreUint32((*uint32)(unsafe.Pointer(a)), v) }
func setRegisterBits(a uintptr, v uint32)   { writeRegister(a, readRegister(a)|v) }
func clearRegisterBits(a uintptr, v uint32) { writeRegister(a, readRegister(a)&^v) }
func writeRegister8(a uintptr, v byte)      { volatile.StoreUint8((*uint8)(unsafe.Pointer(a)), v) }
func readRegister8(a uintptr) byte          { return volatile.LoadUint8((*uint8)(unsafe.Pointer(a))) }
func waitForRegister(a uintptr, mask uint32, on bool) bool {
	for i := 0; i < 100000; i++ {
		if (readRegister(a)&mask != 0) == on {
			return true
		}
	}
	return false
}
func delay(ms uint32) {
	writeRegister(0xe000e014, 15999)
	writeRegister(0xe000e018, 0)
	writeRegister(0xe000e010, 5)
	for ; ms > 0; ms-- {
		if !waitForRegister(0xe000e010, 1<<16, true) {
			writeRegister(0xe000e010, 0)
			sensorError()
		}
	}
	writeRegister(0xe000e010, 0)
}

type board struct{}

func (board) Cycles() uint32                { return readRegister(0xe0001004) }
func (board) Seed() uint32                  { return calibrationSeed }
func (board) ButtonPressed() bool           { return readRegister(0x40020010)&1 != 0 }
func (board) GyroRates() (int, int, bool)   { return gyroRates() }
func (board) Rect(x, y, w, h int, c uint16) { lcdRect(x, y, w, h, c) }
func (board) Delay(ms uint32)               { delay(ms) }
func (board) SensorError()                  { sensorError() }
func sensorError() {
	setRegisterBits(rcc+0x30, 1<<6)
	_ = readRegister(rcc + 0x30)
	clearRegisterBits(0x40021804, 1<<14)
	writeRegister(0x40021800, (readRegister(0x40021800)&^(3<<28))|(1<<28))
	writeRegister(0x40021818, 1<<14)
	lcdRect(0, 0, 320, 240, 0xf800)
	for {
	}
}
func main() {
	// Switch TinyGo's startup clock back to the 16 MHz internal oscillator.
	setRegisterBits(rcc, 1)
	if !waitForRegister(rcc, 2, true) {
		sensorError()
	}
	clearRegisterBits(rcc+8, 3) // Select HSI as the system clock.
	if !waitForRegister(rcc+8, 12, false) {
		sensorError()
	}
	clearRegisterBits(rcc, 1<<24) // Stop the main PLL before changing its configuration.
	if !waitForRegister(rcc, 1<<25, false) {
		sensorError()
	}
	clearRegisterBits(rcc+8, (15<<4)|(7<<10)|(7<<13))
	setRegisterBits(rcc+0x30, 1|(1<<2))
	_ = readRegister(rcc + 0x30)
	clearRegisterBits(0x40020000, 3)
	clearRegisterBits(0x4002000c, 3)
	writeRegister(pc+0x18, 1<<1)
	clearRegisterBits(pc+4, 1<<1)
	writeRegister(pc, (readRegister(pc)&^(3<<2))|(1<<2))
	if !lcdInit() {
		sensorError()
	}
	game.ShowCalibration(board{})
	delay(1500) // Give the player time to place the board flat.
	if !gyroStart() {
		sensorError()
	}
	setRegisterBits(0xe000edfc, 1<<24)
	writeRegister(0xe0001004, 0)
	setRegisterBits(0xe0001000, 1)
	if (board{}).ButtonPressed() {
		game.RunSimulation(board{})
	} else {
		game.Run(board{})
	}
}
