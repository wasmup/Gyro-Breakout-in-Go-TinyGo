//go:build tinygo && stm32f4disco

package main

var filter gyroFilter
var calibrationSeed uint32

func transfer(out byte) (byte, bool) {
	if !waitForRegister(spi5+8, 2, true) {
		return 0, false
	}
	writeRegister8(spi5+0xc, out)
	if !waitForRegister(spi5+8, 1, true) {
		return 0, false
	}
	return readRegister8(spi5 + 0xc), true
}
func transaction(command byte, data []byte, read bool) bool {
	writeRegister(pc+0x18, 1<<17)
	_, ok := transfer(command)
	for i := 0; ok && i < len(data); i++ {
		v := data[i]
		if read {
			v = 0xff
		}
		var in byte
		in, ok = transfer(v)
		if read {
			data[i] = in
		}
	}
	if ok {
		ok = waitForRegister(spi5+8, 1<<7, false)
	}
	writeRegister(pc+0x18, 1<<1)
	return ok
}
func readAxes() ([3]int16, bool) {
	var result [3]int16
	var data [6]byte
	if !transaction(0xe8, data[:], true) {
		return result, false
	}
	for i := 0; i < 3; i++ {
		result[i] = int16(uint16(data[2*i]) | uint16(data[2*i+1])<<8)
	}
	return result, true
}
func gyroStart() bool {
	writeRegister(pc+0x18, (1<<1)|(1<<2))
	writeRegister(pc, (readRegister(pc)&^(3<<2))|(1<<2))
	clearRegisterBits(pc+4, 1<<1)
	writeRegister(pf, (readRegister(pf)&^(3<<16))|(2<<16))
	writeRegister(pf+0x24, (readRegister(pf+0x24)&^15)|5)
	_ = readRegister(spi5 + 0xc)
	_ = readRegister(spi5 + 8)
	var id [1]byte
	if !transaction(0x8f, id[:], true) || (id[0] != 0xd4 && id[0] != 0xd7) {
		return false
	}
	control := [1]byte{0x0f}
	if !transaction(0x20, control[:], false) {
		return false
	}
	control[0] = 0x80
	if !transaction(0x23, control[:], false) {
		return false
	}
	delay(200)
	filter = gyroFilter{}
	calibrationSeed = 2166136261
	for i := 0; i < 32; i++ {
		axes, ok := readAxes()
		if !ok {
			return false
		}
		for j := 0; j < 3; j++ {
			filter.offsets[j] += int32(axes[j])
			calibrationSeed = (calibrationSeed ^ uint32(uint16(axes[j]))) * 16777619
		}
		delay(12)
	}
	for i := 0; i < 3; i++ {
		filter.offsets[i] /= 32
	}
	return true
}
func gyroRates() (int, int, bool) {
	axes, ok := readAxes()
	if !ok {
		return 0, 0, false
	}
	rates := filter.rates(axes)
	return rates[0], rates[1], true
}
