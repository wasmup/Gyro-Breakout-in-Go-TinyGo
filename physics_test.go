package game

import "testing"

func TestControls(t *testing.T) {
	position, fraction := 100, 0
	movePaddle(&position, &fraction, 2)
	if position != 100 {
		t.Fatal(position)
	}
	movePaddle(&position, &fraction, -40)
	if position != 115 {
		t.Fatal(position)
	}
	position = screenW - paddleW - 1
	movePaddle(&position, &fraction, -100)
	if position != screenW-paddleW || fraction != 0 {
		t.Fatal(position, fraction)
	}
	speed, ready, last := 5, true, uint32(0)
	adjustSpeed(&speed, &ready, &last, -50, cyclesPerSecond)
	if speed != 6 || ready {
		t.Fatal(speed, ready)
	}
	adjustSpeed(&speed, &ready, &last, -50, 2*cyclesPerSecond)
	if speed != 6 {
		t.Fatal(speed)
	}
	adjustSpeed(&speed, &ready, &last, 0, 2*cyclesPerSecond)
	adjustSpeed(&speed, &ready, &last, 50, 2*cyclesPerSecond)
	if speed != 5 {
		t.Fatal(speed)
	}
}
func TestCollision(t *testing.T) {
	hit, _, _ := paddleCollision(88, 203, 91, 205, 10, 100, 214, 96)
	if hit {
		t.Fatal("false hit")
	}
	hit, _, impact := paddleCollision(91.5, 203.2, 88.5, 205.2, 10, 100, 214, 96)
	if !hit || impact <= 90 || impact >= 91.5 {
		t.Fatal(hit, impact)
	}
	x, y := brickEntry(28.8, 29.8, 2, 2, 10, 40, 40, 38, 12)
	if x <= y {
		t.Fatal(x, y)
	}
	x, y = brickEntry(29.8, 28.8, 2, 2, 10, 40, 40, 38, 12)
	if y <= x {
		t.Fatal(x, y)
	}
}

func TestGameStateLaunchAndLifeLoss(t *testing.T) {
	var s gameState
	s.init(42, cyclesPerSecond, false)
	if s.remaining != 40 || s.lives != 5 || s.playing {
		t.Fatal(s)
	}
	s.step(0, 0, true, cyclesPerSecond)
	if !s.playing {
		t.Fatal("button did not launch ball")
	}
	s.ballY = screenH
	s.vy = 5
	s.step(0, 0, false, cyclesPerSecond+1)
	if s.lives != 4 || s.playing || s.finished {
		t.Fatal(s.lives, s.playing, s.finished)
	}
}

func TestPaddleBounceRemainsWithinWalls(t *testing.T) {
	for _, tc := range []struct {
		name         string
		paddle       int
		x, vx        float32
		wantPositive bool
	}{
		{"left", 0, 0.5, -3, true},
		{"right", screenW - paddleW, screenW - ballSize - 0.5, 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s gameState
			s.init(1, 0, false)
			s.playing, s.paddle = true, tc.paddle
			s.ballX, s.ballY, s.vx, s.vy = tc.x, 202.6, tc.vx, 5
			s.updateBall()
			if s.ballX < 0 || s.ballX > screenW-ballSize {
				t.Fatalf("ball escaped playfield: x=%v", s.ballX)
			}
			if (s.vx > 0) != tc.wantPositive || s.vy >= 0 {
				t.Fatalf("wrong bounce direction: vx=%v vy=%v", s.vx, s.vy)
			}
		})
	}
}

func TestImmediateRelaunchAfterLifeLoss(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.playing = true
	s.ballY, s.vy = screenH, 5
	s.step(0, 0, false, 0)
	if s.lives != 4 || s.playing || s.ballY != paddleY-ballSize {
		t.Fatalf("after miss: lives=%d playing=%t ballY=%v", s.lives, s.playing, s.ballY)
	}
	s.step(0, 0, true, cyclesPerSecond/100)
	if s.lives != 4 || !s.playing || s.ballY >= paddleY-ballSize {
		t.Fatalf("after relaunch: lives=%d playing=%t ballY=%v", s.lives, s.playing, s.ballY)
	}
}

func TestButtonDebounceAcrossCycleWrap(t *testing.T) {
	var b buttonDebouncer
	start := uint32(0xffff0000)
	if b.update(true, start) || b.update(false, start+100) || b.update(true, start+200) {
		t.Fatal("accepted an unstable press")
	}
	if b.update(true, start+200+cyclesPerSecond/50-1) {
		t.Fatal("accepted press too early")
	}
	if !b.update(true, start+200+cyclesPerSecond/50) {
		t.Fatal("did not accept stable press across counter wrap")
	}
}

func TestLastBrickShotWinsWithoutBonus(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.bricks = [rows][cols]bool{}
	s.bricks[0][0] = true
	s.damaged[0][0] = true
	s.remaining, s.bonusBrick, s.playing = 1, 0, true
	s.shotActive[0], s.shotX[0], s.shotY[0] = true, 1, brickTop+brickHeight
	s.updateShots()
	if !s.finished || s.remaining != 0 || s.lives != 5 || s.shotActive[0] {
		t.Fatalf("unexpected state after final shot: remaining=%d lives=%d finished=%t", s.remaining, s.lives, s.finished)
	}
}

func TestFinalLifeEndsGame(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.lives, s.playing, s.ballY, s.vy = 1, true, screenH, 5
	s.step(0, 0, false, 0)
	if s.lives != 0 || !s.finished || s.playing {
		t.Fatalf("unexpected state after final miss: lives=%d finished=%t playing=%t", s.lives, s.finished, s.playing)
	}
}

func TestBallBonusAndWin(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.bricks = [rows][cols]bool{}
	s.bricks[0][0] = true
	s.damaged[0][0] = true
	s.remaining, s.bonusBrick, s.playing = 1, 0, true
	s.ballX, s.ballY, s.vx, s.vy = 5, brickTop+brickHeight+0.5, 0, -5
	s.step(0, 0, false, 0)
	if !s.finished || s.remaining != 0 || s.lives != 7 || s.vy <= 0 {
		t.Fatalf("final bonus brick: remaining=%d lives=%d finished=%t vy=%v", s.remaining, s.lives, s.finished, s.vy)
	}
}

func TestFireRepeatAndRelease(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.controls(0, 0, true, 0) // Launch, without firing.
	if !s.playing || s.shotActive[0] {
		t.Fatal("launch also fired")
	}
	s.controls(0, 0, true, cyclesPerSecond/5-1)
	if s.shotActive[0] {
		t.Fatal("fired before repeat interval")
	}
	s.controls(0, 0, true, cyclesPerSecond/5)
	if !s.shotActive[0] || !s.shotActive[1] {
		t.Fatal("holding button did not fire pair")
	}
	s.controls(0, 0, false, cyclesPerSecond/5+1)
	s.controls(0, 0, true, cyclesPerSecond/5+2)
	if !s.shotActive[2] || !s.shotActive[3] {
		t.Fatal("new press did not fire pair")
	}
}

func TestInitResetsFinishedGameAndRequiresRelease(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.finished, s.playing, s.lives, s.remaining = true, true, 0, 0
	s.shotActive[0] = true
	s.init(2, 123, true)
	if s.finished || s.playing || s.lives != 5 || s.remaining != rows*cols || s.shotActive[0] {
		t.Fatal("initialization did not reset the game")
	}
	s.controls(0, 0, true, 123)
	if s.playing {
		t.Fatal("held restart button launched the ball")
	}
	s.controls(0, 0, false, 124)
	s.controls(0, 0, true, 125)
	if !s.playing {
		t.Fatal("new press did not launch the ball")
	}
}

func TestRoundClockAcrossCounterWrap(t *testing.T) {
	clock := roundClock{last: 0xff000000}
	now := clock.last
	now += cyclesPerSecond
	clock.update(now)
	now += cyclesPerSecond
	clock.update(now)
	if got := clock.seconds(); got != 2 {
		t.Fatalf("elapsed = %d seconds, want 2", got)
	}
}

func TestBrickRequiresTwoBallHits(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.bricks = [rows][cols]bool{}
	s.bricks[0][0] = true
	s.remaining, s.bonusBrick = 1, -1
	for hit := 1; hit <= 2; hit++ {
		s.ballX, s.ballY, s.vy = 5, 46, -5
		count := 0
		s.resolveBricks(5, 48, &count)
		if count != 1 {
			t.Fatalf("hit %d: collisions=%d", hit, count)
		}
		if hit == 1 && (!s.bricks[0][0] || !s.damaged[0][0] || s.remaining != 1) {
			t.Fatalf("first hit: alive=%t damaged=%t remaining=%d", s.bricks[0][0], s.damaged[0][0], s.remaining)
		}
	}
	if s.bricks[0][0] || s.remaining != 0 {
		t.Fatal("second ball hit did not destroy brick")
	}
}

func TestMixedHitsAwardBonusOnlyWhenBallDestroys(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.bricks = [rows][cols]bool{}
	s.bricks[0][0] = true
	s.remaining, s.bonusBrick, s.playing = 1, 0, true
	s.shotActive[0], s.shotX[0], s.shotY[0] = true, 1, brickTop+brickHeight
	s.updateShots()
	if !s.damaged[0][0] || !s.bricks[0][0] || s.lives != 5 || s.finished {
		t.Fatal("first fireball hit should only damage bonus brick")
	}
	s.ballX, s.ballY, s.vy = 5, 46, -5
	hits := 0
	s.resolveBricks(5, 48, &hits)
	if s.bricks[0][0] || s.remaining != 0 || s.lives != 7 {
		t.Fatalf("bonus destruction: remaining=%d lives=%d", s.remaining, s.lives)
	}
}
