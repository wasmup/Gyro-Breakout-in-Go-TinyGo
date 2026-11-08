package game

const (
	screenH      = 240
	paddleY      = 214
	ballSize     = 10
	shotCount    = 16
	rows         = 5
	cols         = 8
	brickPitchX  = 40
	brickPitchY  = 15
	brickTop     = 35
	brickWidth   = 38
	brickHeight  = 12
	playfieldTop = 18
	maxLives     = 7
)

type gameState struct {
	bricks     [rows][cols]bool
	damaged    [rows][cols]bool
	remaining  int
	bonusBrick int
	lives      int

	paddle         int
	paddleFraction int

	ballX, ballY float32
	vx, vy       float32
	playing      bool
	finished     bool

	speedTenths       int
	speedGestureReady bool
	previousButton    bool
	lastSpeedGesture  uint32
	lastFire          uint32

	shotActive [shotCount]bool
	shotX      [shotCount]int
	shotY      [shotCount]int
}

func mixRandom(v uint32) uint32 {
	v ^= v >> 16
	v *= 0x7feb352d
	v ^= v >> 15
	v *= 0x846ca68b
	return v ^ (v >> 16)
}
func (s *gameState) init(seed, now uint32, pressed bool) {
	*s = gameState{}
	for r := range s.bricks {
		for c := range s.bricks[r] {
			s.bricks[r][c] = true
		}
	}
	s.remaining = rows * cols
	s.lives = 5
	s.paddle = (screenW - paddleW) / 2
	s.bonusBrick = int(mixRandom(now^seed) % (rows * cols))
	s.ballX = (screenW - ballSize) / 2
	s.ballY = paddleY - ballSize
	s.vx = 3
	s.vy = -5
	s.speedTenths = 5
	s.speedGestureReady = true
	s.previousButton = pressed
	s.lastSpeedGesture = now - cyclesPerSecond
	s.lastFire = now
}
func (s *gameState) controls(paddleRate, speedRate int, pressed bool, now uint32) {
	movePaddle(&s.paddle, &s.paddleFraction, paddleRate)
	adjustSpeed(&s.speedTenths, &s.speedGestureReady, &s.lastSpeedGesture, speedRate, now)
	if pressed && !s.previousButton && !s.playing {
		s.playing = true
		s.lastFire = now
	} else if pressed && s.playing && (!s.previousButton || now-s.lastFire >= cyclesPerSecond/5) {
		for i := 0; i < shotCount; i += 2 {
			if !s.shotActive[i] && !s.shotActive[i+1] {
				s.shotX[i] = s.paddle
				s.shotX[i+1] = s.paddle + paddleW - ballSize
				s.shotY[i] = paddleY - ballSize
				s.shotY[i+1] = s.shotY[i]
				s.shotActive[i] = true
				s.shotActive[i+1] = true
				s.lastFire = now
				break
			}
		}
	}
	s.previousButton = pressed
}
func (s *gameState) resolveBricks(previousX, previousY float32, hits *int) {
	for *hits < 4 {
		dx, dy := s.ballX-previousX, s.ballY-previousY
		firstEntry, entryX, entryY := float32(2), float32(-1), float32(-1)
		hitRow, hitCol := -1, -1
		for r := range rows {
			for c := range cols {
				x, y := c*brickPitchX+1, brickTop+r*brickPitchY
				if !s.bricks[r][c] || s.ballX+ballSize <= float32(x) || s.ballX >= float32(x+brickWidth) || s.ballY+ballSize <= float32(y) || s.ballY >= float32(y+brickHeight) {
					continue
				}
				candidateX, candidateY := brickEntry(previousX, previousY, dx, dy, ballSize, x, y, brickWidth, brickHeight)
				entry := candidateX
				if candidateY > entry {
					entry = candidateY
				}
				if entry < 0 {
					entry = 0
				}
				if entry < firstEntry {
					firstEntry, hitRow, hitCol, entryX, entryY = entry, r, c, candidateX, candidateY
				}
			}
		}
		if hitRow < 0 {
			break
		}
		impactX, impactY := previousX+dx*firstEntry, previousY+dy*firstEntry
		if s.hitBrick(hitRow, hitCol) && hitRow*cols+hitCol == s.bonusBrick {
			s.lives += 2
		}
		if entryX >= 0 && entryX > entryY {
			s.ballX = previousX + dx*(2*entryX-1)
			s.vx = -s.vx
		} else if entryY >= 0 {
			s.ballY = previousY + dy*(2*entryY-1)
			s.vy = -s.vy
		} else {
			s.ballY = previousY
			s.vy = -s.vy
		}
		*hits++
		previousX, previousY = impactX, impactY
		if s.ballX < 0 {
			s.ballX = -s.ballX
			s.vx = -s.vx
		}
		if s.ballX > screenW-ballSize {
			s.ballX = 2*(screenW-ballSize) - s.ballX
			s.vx = -s.vx
		}
		if s.ballY < playfieldTop {
			s.ballY = 2*playfieldTop - s.ballY
			s.vy = -s.vy
		}
	}
}
func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
func (s *gameState) updateBall() {
	if !s.playing {
		s.ballX = float32(s.paddle + (paddleW-ballSize)/2)
		s.ballY = paddleY - ballSize
		return
	}
	scale := float32(s.speedTenths) * 0.1
	velocity := abs(s.vx)
	if abs(s.vy) > velocity {
		velocity = abs(s.vy)
	}
	steps := int(velocity*scale/2) + 1
	hits := 0
	for step := 0; step < steps && hits < 4; step++ {
		px, py := s.ballX, s.ballY
		s.ballX += s.vx * scale / float32(steps)
		s.ballY += s.vy * scale / float32(steps)
		if s.ballX <= 0 {
			s.ballX = -s.ballX
			s.vx = -s.vx
		}
		if s.ballX >= screenW-ballSize {
			s.ballX = 2*(screenW-ballSize) - s.ballX
			s.vx = -s.vx
		}
		if s.ballY <= playfieldTop {
			s.ballY = 2*playfieldTop - s.ballY
			s.vy = -s.vy
		}
		hit, f, ix := paddleCollision(px, py, s.ballX, s.ballY, ballSize, s.paddle, paddleY, paddleW)
		if s.vy > 0 && hit {
			s.vx = (ix + ballSize/2 - float32(s.paddle+paddleW/2)) / 5
			s.vy = -5
			s.ballX = ix + s.vx*scale/float32(steps)*(1-f)
			s.ballY = paddleY - ballSize + s.vy*scale/float32(steps)*(1-f)
			// The paddle bounce can send the remaining movement through a wall.
			if s.ballX < 0 {
				s.ballX = -s.ballX
				s.vx = -s.vx
			} else if s.ballX > screenW-ballSize {
				s.ballX = 2*(screenW-ballSize) - s.ballX
				s.vx = -s.vx
			}
		}
		s.resolveBricks(px, py, &hits)
	}
	if s.ballY >= screenH {
		s.lives--
		s.playing = false
		s.ballX = float32(s.paddle + (paddleW-ballSize)/2)
		s.ballY = paddleY - ballSize
		s.vx = 3
		s.vy = -5
		if s.lives == 0 {
			s.finished = true
		}
	}
	if s.remaining == 0 {
		s.finished = true
	}
}

// hitBrick returns true only when the second hit destroys the brick.
func (s *gameState) hitBrick(row, col int) bool {
	if !s.damaged[row][col] {
		s.damaged[row][col] = true
		return false
	}
	s.bricks[row][col] = false
	s.remaining--
	return true
}

func (s *gameState) updateShots() {
	for i := range shotCount {
		if !s.shotActive[i] {
			continue
		}
		previousY := s.shotY[i]
		s.shotY[i] -= 8
		if !s.playing || s.shotY[i] < playfieldTop {
			s.shotActive[i] = false
			continue
		}
		firstCol := clamp(s.shotX[i]/brickPitchX, 0, cols-1)
		lastCol := clamp((s.shotX[i]+ballSize-1)/brickPitchX, 0, cols-1)
		firstRow := clamp((s.shotY[i]-brickTop)/brickPitchY, 0, rows-1)
		lastRow := clamp((previousY+ballSize-1-brickTop)/brickPitchY, 0, rows-1)
		for r := lastRow; r >= firstRow && s.shotActive[i]; r-- {
			for c := firstCol; c <= lastCol; c++ {
				x, y := c*brickPitchX+1, brickTop+r*brickPitchY
				if s.bricks[r][c] && s.shotX[i]+ballSize > x && s.shotX[i] < x+brickWidth && s.shotY[i] < y+brickHeight && previousY+ballSize > y {
					s.hitBrick(r, c)
					s.shotActive[i] = false
					if s.remaining == 0 {
						s.finished = true
					}
					break
				}
			}
		}
	}
}
func (s *gameState) step(paddleRate, speedRate int, pressed bool, now uint32) {
	if s.finished {
		return
	}
	s.controls(paddleRate, speedRate, pressed, now)
	s.updateBall()
	s.updateShots()
}
