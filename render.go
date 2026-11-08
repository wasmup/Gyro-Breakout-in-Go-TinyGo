package game

import "strconv"

const (
	black uint16 = 0x0000
	white uint16 = 0xffff
	red   uint16 = 0xf800
	green uint16 = 0x07e0
	blue  uint16 = 0x001f
)

// Platform supplies the hardware services required by the game. Cycles must
// advance at 16 MHz and wrap naturally at 32 bits. GyroRates returns integer
// degrees per second, Rect uses RGB565 colors, and Delay takes milliseconds.
type Platform interface {
	Cycles() uint32
	Seed() uint32
	ButtonPressed() bool
	GyroRates() (paddle, speed int, ok bool)
	Rect(x, y, width, height int, color uint16)
	Delay(ms uint32)
	SensorError()
}

type renderer struct{ p Platform }

func brickColor(s *gameState, row, col int) uint16 {
	if s.damaged[row][col] {
		return red
	}
	if row*cols+col == s.bonusBrick {
		return blue
	}
	return green
}

func (r renderer) rect(x, y, w, h int, color uint16) {
	if x >= 0 && y >= 0 && w > 0 && h > 0 {
		r.p.Rect(x, y, w, h, color)
	}
}
func (r renderer) eraseBall(ox, oy, nx, ny int) {
	left, right := clamp(nx-ox, 0, ballSize), clamp(ox-nx, 0, ballSize)
	top, bottom := clamp(ny-oy, 0, ballSize), clamp(oy-ny, 0, ballSize)
	r.rect(ox, oy, ballSize, top, black)
	r.rect(ox, oy+ballSize-bottom, ballSize, bottom, black)
	r.rect(ox, oy+top, left, ballSize-top-bottom, black)
	r.rect(ox+ballSize-right, oy+top, right, ballSize-top-bottom, black)
}
func (r renderer) restore(s *gameState, x, y int) {
	if y+ballSize <= brickTop || y >= brickTop+rows*brickPitchY || x >= screenW {
		return
	}
	fc, lc := clamp(x/brickPitchX, 0, cols-1), clamp((x+ballSize-1)/brickPitchX, 0, cols-1)
	fr, lr := clamp((y-brickTop)/brickPitchY, 0, rows-1), clamp((y+ballSize-1-brickTop)/brickPitchY, 0, rows-1)
	for row := fr; row <= lr; row++ {
		for col := fc; col <= lc; col++ {
			if s.bricks[row][col] {
				r.rect(col*brickPitchX+1, brickTop+row*brickPitchY, brickWidth, brickHeight, brickColor(s, row, col))
			}
		}
	}
}
func (r renderer) speed(t int) {
	glyphs := [...]uint16{0x7b6f, 0x2492, 0x73e7, 0x73cf, 0x5bc9, 0x79cf, 0x79ef, 0x7249, 0x7bef, 0x7bcf, 0x0002, 0x0550}
	symbols := [...]int{t / 10, 10, t % 10, 11}
	r.rect(screenW-36, 3, 34, 10, black)
	for i, s := range symbols {
		for y := range 5 {
			for x := range 3 {
				if glyphs[s]&(1<<uint(14-y*3-x)) != 0 {
					r.rect(screenW-36+i*8+x*2, 3+y*2, 2, 2, white)
				}
			}
		}
	}
}

// letters contains the 5×7 glyphs used by the game's messages.
var letters = [26][7]uint8{
	'A' - 'A': {14, 17, 17, 31, 17, 17, 17},
	'B' - 'A': {30, 17, 17, 30, 17, 17, 30},
	'C' - 'A': {14, 17, 16, 16, 16, 17, 14},
	'D' - 'A': {30, 17, 17, 17, 17, 17, 30},
	'E' - 'A': {31, 16, 16, 30, 16, 16, 31},
	'F' - 'A': {31, 16, 16, 30, 16, 16, 16},
	'G' - 'A': {14, 17, 16, 23, 17, 17, 15},
	'I' - 'A': {14, 4, 4, 4, 4, 4, 14},
	'K' - 'A': {17, 18, 20, 24, 20, 18, 17},
	'L' - 'A': {16, 16, 16, 16, 16, 16, 31},
	'M' - 'A': {17, 27, 21, 21, 17, 17, 17},
	'N' - 'A': {17, 25, 25, 21, 19, 19, 17},
	'O' - 'A': {14, 17, 17, 17, 17, 17, 14},
	'P' - 'A': {30, 17, 17, 30, 16, 16, 16},
	'R' - 'A': {30, 17, 17, 30, 20, 18, 17},
	'T' - 'A': {31, 4, 4, 4, 4, 4, 4},
	'U' - 'A': {17, 17, 17, 17, 17, 17, 14},
	'W' - 'A': {17, 17, 17, 21, 21, 21, 10},
	'X' - 'A': {17, 17, 10, 4, 10, 17, 17},
	'Y' - 'A': {17, 17, 10, 4, 4, 4, 4},
}

var digits = [10][7]uint8{
	{14, 17, 19, 21, 25, 17, 14},
	{4, 12, 4, 4, 4, 4, 14},
	{14, 17, 1, 2, 4, 8, 31},
	{30, 1, 1, 14, 1, 1, 30},
	{2, 6, 10, 18, 31, 2, 2},
	{31, 16, 16, 30, 1, 1, 30},
	{14, 16, 16, 30, 17, 17, 14},
	{31, 1, 2, 4, 8, 8, 8},
	{14, 17, 17, 14, 17, 17, 14},
	{14, 17, 17, 15, 1, 1, 14},
}

func (r renderer) text(text string, top, scale int, color uint16) {
	left := (screenW - (len(text)*6-1)*scale) / 2
	for i := 0; i < len(text); i++ {
		var glyph [7]uint8
		switch {
		case text[i] >= 'A' && text[i] <= 'Z':
			glyph = letters[text[i]-'A']
		case text[i] >= '0' && text[i] <= '9':
			glyph = digits[text[i]-'0']
		case text[i] == ':':
			glyph = [7]uint8{0, 4, 4, 0, 4, 4, 0}
		default:
			continue
		}
		for y, row := range glyph {
			for x := range 5 {
				if row&(1<<uint(4-x)) != 0 {
					r.rect(left+i*6*scale+x*scale, top+y*scale, scale, scale, color)
				}
			}
		}
	}
}

func (r renderer) message(win bool) {
	// Clear the message area so the text remains legible over moving objects.
	r.rect(0, 109, screenW, 21, black)
	if win {
		r.text("YOU WIN", 109, 3, green)
	} else {
		r.text("MAYBE NEXT TIME", 109, 2, red)
	}
}

func formatElapsed(seconds uint64) string {
	minutes := strconv.FormatUint(seconds/60, 10)
	if len(minutes) < 2 {
		minutes = "0" + minutes
	}
	remainder := strconv.FormatUint(seconds%60, 10)
	if len(remainder) < 2 {
		remainder = "0" + remainder
	}
	return "TIME " + minutes + ":" + remainder
}

func (r renderer) elapsedTime(seconds uint64) {
	r.rect(0, 140, screenW, 14, black)
	r.text(formatElapsed(seconds), 140, 2, white)
}

// ShowCalibration asks the player to keep the board flat during gyro setup.
func ShowCalibration(p Platform) {
	r := renderer{p}
	r.rect(0, 0, screenW, screenH, black)
	r.text("KEEP BOARD FLAT", 100, 2, white)
	r.text("CALIBRATING", 124, 2, white)
}
