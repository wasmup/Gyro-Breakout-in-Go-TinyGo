package game

// Run executes the game using the supplied board services. SensorError must
// stop execution when a sensor fails.
func Run(p Platform) {
	run(p, false)
}

// RunSimulation plays repeated automatic games until the board is reset.
func RunSimulation(p Platform) {
	run(p, true)
}

func run(p Platform, simulation bool) {
	r := renderer{p}
	var input buttonDebouncer
	button := func() bool { return input.update(p.ButtonPressed(), p.Cycles()) }
	previous := button()
	previousAimOffset := 4 // Outside the range of possible offsets.
	for {
		var s gameState
		if simulation {
			previous = false
		}
		start := p.Cycles()
		seed := p.Seed()
		s.init(seed, start, previous)
		controller := newSimulationController(seed^start, previousAimOffset)
		previousAimOffset = controller.aimOffset
		clock := roundClock{last: start}
		if simulation {
			s.speedTenths = simulationSpeedTenths
		}
		r.rect(0, 0, screenW, screenH, black)
		for row := range rows {
			for col := range cols {
				r.rect(col*brickPitchX+1, brickTop+row*brickPitchY, brickWidth, brickHeight, brickColor(&s, row, col))
			}
		}
		displayed := -1
		next := p.Cycles()
		for !s.finished {
			old := s
			paddle, speed, ok := p.GyroRates()
			if !ok {
				p.SensorError()
				return
			}
			var pressed bool
			if simulation {
				paddle, pressed = controller.input(&s)
				physicalButton := button()
				if s.playing {
					pressed = physicalButton
				}
			} else {
				pressed = button()
			}
			s.step(paddle, speed, pressed, p.Cycles())
			clock.update(p.Cycles())
			displayed = r.drawFrame(&old, &s, displayed)
			next += frameCycles
			now := p.Cycles()
			if int32(next-now) > 0 {
				for int32(next-p.Cycles()) > 0 {
				}
			} else if now-next > 3*frameCycles {
				next = now - 3*frameCycles
			}
		}
		if s.remaining == 0 {
			r.rect(screenW/2-5, 4, 10, 8, green)
		}
		r.message(s.remaining == 0)
		r.elapsedTime(clock.seconds())
		if simulation {
			p.Delay(2000) // Leave the result visible before starting the next game.
			continue
		}
		for {
			previous = button()
			p.Delay(20)
			if !previous {
				break
			}
		}
		for {
			previous = button()
			p.Delay(20)
			if previous {
				break
			}
		}
	}
}

// buttonDebouncer accepts an input edge only after it remains stable for 20 ms.
type buttonDebouncer struct {
	stable, candidate bool
	changed           uint32
}

func (b *buttonDebouncer) update(raw bool, now uint32) bool {
	if raw != b.candidate {
		b.candidate = raw
		b.changed = now
	}
	if b.candidate != b.stable && now-b.changed >= cyclesPerSecond/50 {
		b.stable = b.candidate
	}
	return b.stable
}

// drawFrame updates only the portions of the display changed by a game step.
func (r renderer) drawFrame(old, s *gameState, displayed int) int {
	if s.speedTenths != displayed {
		r.speed(s.speedTenths)
		displayed = s.speedTenths
	}
	for row := range rows {
		for col := range cols {
			if old.bricks[row][col] && !s.bricks[row][col] {
				r.rect(col*brickPitchX+1, brickTop+row*brickPitchY, brickWidth, brickHeight, black)
			} else if s.bricks[row][col] && old.damaged[row][col] != s.damaged[row][col] {
				r.rect(col*brickPitchX+1, brickTop+row*brickPitchY, brickWidth, brickHeight, brickColor(s, row, col))
			}
		}
	}
	for i := range shotCount {
		if old.shotActive[i] {
			r.rect(old.shotX[i], old.shotY[i], ballSize, ballSize, black)
			r.restore(s, old.shotX[i], old.shotY[i])
		}
	}
	r.eraseBall(int(old.ballX), int(old.ballY), int(s.ballX), int(s.ballY))
	r.restore(s, int(old.ballX), int(old.ballY))
	if s.paddle > old.paddle {
		r.rect(old.paddle, paddleY, s.paddle-old.paddle, 7, black)
	} else if s.paddle < old.paddle {
		r.rect(s.paddle+paddleW, paddleY, old.paddle-s.paddle, 7, black)
	}
	for i := range maxLives {
		color := black
		if i < s.lives {
			color = blue
		}
		r.rect(5+i*14, 4, 10, 8, color)
	}
	r.rect(s.paddle, paddleY, paddleW, 7, white)
	if s.ballY < screenH {
		r.rect(int(s.ballX), int(s.ballY), ballSize, ballSize, green)
	}
	for i := range shotCount {
		if s.shotActive[i] {
			r.rect(s.shotX[i], s.shotY[i], ballSize, ballSize, red)
		}
	}
	return displayed
}

// roundClock accumulates unsigned differences so the 32-bit cycle counter may wrap.
type roundClock struct {
	last   uint32
	cycles uint64
}

func (c *roundClock) update(now uint32) {
	c.cycles += uint64(now - c.last)
	c.last = now
}

func (c *roundClock) seconds() uint64 { return c.cycles / uint64(cyclesPerSecond) }
