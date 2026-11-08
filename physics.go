package game

const (
	screenW                = 320
	paddleW                = 96
	cyclesPerSecond uint32 = 16000000
	frameCycles            = cyclesPerSecond / 100
)

func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
func movePaddle(position, fraction *int, rate int) {
	if rate >= -2 && rate <= 2 {
		rate = 0
	}
	gain := clamp(rate, -100, 100)
	if gain > 20 {
		gain += gain - 20
	} else if gain < -20 {
		gain += gain + 20
	}
	*fraction -= gain
	*position = clamp(*position+*fraction/4, 0, screenW-paddleW)
	*fraction %= 4
	if *position == 0 || *position == screenW-paddleW {
		*fraction = 0
	}
}
func adjustSpeed(speed *int, ready *bool, last *uint32, rate int, now uint32) {
	if rate >= -10 && rate <= 10 {
		*ready = true
	}
	if *ready && (rate >= 40 || rate <= -40) {
		*ready = false
		if now-*last >= cyclesPerSecond {
			delta := -1
			if rate < 0 {
				delta = 1
			}
			*speed = clamp(*speed+delta, 5, 13)
			*last = now
		}
	}
}
func paddleCollision(oldX, oldY, newX, newY float32, size, paddleX, paddleY, paddleWidth int) (bool, float32, float32) {
	if newY <= oldY || oldY+float32(size) > float32(paddleY) || newY+float32(size) < float32(paddleY) {
		return false, 0, 0
	}
	fraction := (float32(paddleY-size) - oldY) / (newY - oldY)
	impact := oldX + (newX-oldX)*fraction
	return impact+float32(size) > float32(paddleX) && impact < float32(paddleX+paddleWidth), fraction, impact
}
func brickEntry(oldX, oldY, dx, dy float32, size, x, y, width, height int) (float32, float32) {
	ex, ey := float32(-1), float32(-1)
	if dx > 0 && oldX+float32(size) <= float32(x) {
		ex = (float32(x) - oldX - float32(size)) / dx
	} else if dx < 0 && oldX >= float32(x+width) {
		ex = (float32(x+width) - oldX) / dx
	}
	if dy > 0 && oldY+float32(size) <= float32(y) {
		ey = (float32(y) - oldY - float32(size)) / dy
	} else if dy < 0 && oldY >= float32(y+height) {
		ey = (float32(y+height) - oldY) / dy
	}
	return ex, ey
}
