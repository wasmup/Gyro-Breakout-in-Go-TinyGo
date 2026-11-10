package game

const simulationSpeedTenths = 10

// simulationController varies its aim between rounds while predicting the
// next paddle impact and accounting for side-wall reflections.
type simulationController struct {
	aimOffset      int
	reachedTop     bool
	lastBrickHits  uint32
	lastPaddleHits uint32
	missedReturns  int
	shiftStep      int
}

func newSimulationController(seed uint32, previousOffset int) simulationController {
	offset := int(mixRandom(seed)%7) - 3
	if offset == previousOffset {
		offset++
		if offset > 3 {
			offset = -3
		}
	}
	return simulationController{aimOffset: offset}
}

func (controller *simulationController) input(s *gameState) (paddleRate int, pressed bool) {
	if s.brickHits != controller.lastBrickHits {
		controller.missedReturns = 0
		controller.lastBrickHits = s.brickHits
	}
	if s.paddleHits != controller.lastPaddleHits {
		controller.lastPaddleHits = s.paddleHits
		controller.missedReturns++
		if controller.missedReturns >= 2 {
			controller.shiftStep = (controller.shiftStep + 1) % 5
			controller.missedReturns = 0
		}
	}
	impact := s.ballX
	if s.playing && s.vy > 0 {
		impact += s.vx * (float32(paddleY-ballSize) - s.ballY) / s.vy
	}
	const edge = screenW - ballSize
	// Fold the projected position into the playfield after any wall bounces.
	period := float32(2 * edge)
	impact -= float32(int(impact/period)) * period
	if impact < 0 {
		impact += period
	}
	if impact > edge {
		impact = period - impact
	}
	// Open a route through the center, then continue using it until the ball
	// reaches the space above the bricks.
	centerColumn := cols / 2
	centerOpen := true
	for row := range rows {
		if s.bricks[row][centerColumn] {
			centerOpen = false
			break
		}
	}
	if centerOpen && s.ballY+ballSize <= brickTop {
		controller.reachedTop = true
	}
	column := centerColumn
	if controller.reachedTop {
		// Once above the grid, prefer the column closest to being cleared.
		best := rows + 1
		for col := cols - 1; col >= 0; col-- {
			count := 0
			for row := range rows {
				if s.bricks[row][col] {
					count++
				}
			}
			if count > 0 && count < best {
				column, best = col, count
			}
		}
	}
	brickX := column*brickPitchX + 1 + brickWidth/2 + controller.aimOffset
	brickY := brickTop - ballSize
	for row := rows - 1; row >= 0; row-- {
		if s.bricks[row][column] {
			brickY = brickTop + row*brickPitchY + brickHeight
			break
		}
	}
	velocity := (float32(brickX) - (impact + ballSize/2)) * 5 / float32(paddleY-ballSize-brickY)
	// Prefer angled rebounds over near-vertical shots by using more of the
	// paddle's outer area. Keep the chosen direction toward the target.
	const minimumHorizontalSpeed = 4
	if controller.reachedTop {
		if velocity >= 0 && velocity < minimumHorizontalSpeed {
			velocity = minimumHorizontalSpeed
		} else if velocity < 0 && velocity > -minimumHorizontalSpeed {
			velocity = -minimumHorizontalSpeed
		}
	}
	if velocity > 8 {
		velocity = 8
	}
	if velocity < -8 {
		velocity = -8
	}
	// Change the impact point after two returns without a brick hit. Alternate
	// sides and increase the shift so repeated misses try different paths.
	shifts := [...]int{0, 6, -6, 12, -12}
	target := clamp(int(impact+ballSize/2-float32(paddleW/2)-5*velocity)+shifts[controller.shiftStep], 0, screenW-paddleW)
	delta := target - s.paddle
	gain := delta*4 - s.paddleFraction
	if gain > 20 {
		gain = (gain + 20) / 2
	} else if gain < -20 {
		gain = (gain - 20) / 2
	}
	paddleRate = -clamp(gain, -100, 100)
	// Launch automatically; the physical button controls firing during play.
	pressed = !s.playing && !s.previousButton
	return paddleRate, pressed
}
