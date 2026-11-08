package game

import "testing"

func TestSimulationCompletesGames(t *testing.T) {
	for seed := uint32(0); seed < 20; seed++ {
		var s gameState
		s.init(seed, 0, false)
		s.speedTenths = simulationSpeedTenths
		controller := newSimulationController(seed, 4)
		for step := uint32(0); step < 100000 && !s.finished; step++ {
			paddle, pressed := controller.input(&s)
			s.step(paddle, 0, pressed, step*(cyclesPerSecond/100))
			for _, active := range s.shotActive {
				if active {
					t.Fatalf("seed %d fired a shot at step %d", seed, step)
				}
			}
		}
		if !s.finished {
			t.Fatalf("seed %d did not finish: bricks=%d lives=%d", seed, s.remaining, s.lives)
		}
		if s.remaining != 0 {
			t.Errorf("seed %d lost: bricks=%d", seed, s.remaining)
		}
	}
}

type simulationPlatform struct {
	cycles           uint32
	rounds           int
	pauses           int
	gyroReads        int
	speedRate        int
	sawSpeedIncrease bool
	sawSpeedDecrease bool
}

func (p *simulationPlatform) Cycles() uint32    { p.cycles += frameCycles; return p.cycles }
func (*simulationPlatform) Seed() uint32        { return 1 }
func (*simulationPlatform) ButtonPressed() bool { return false }
func (p *simulationPlatform) GyroRates() (int, int, bool) {
	p.gyroReads++
	return 250, p.speedRate, true
}
func (p *simulationPlatform) Rect(x, y, width, height int, color uint16) {
	// The middle pixel in this row distinguishes the 1.1× digit from 1.0×.
	if x == 302 && y == 5 && width == 2 && height == 2 && color == white {
		p.sawSpeedIncrease = true
	}
	if x == 302 && y == 7 && width == 2 && height == 2 && color == white {
		p.sawSpeedDecrease = true
	}
	if x == 0 && y == 0 && width == screenW && height == screenH && color == black {
		p.rounds++
		if p.rounds == 2 {
			panic("second round started")
		}
	}
}
func (p *simulationPlatform) Delay(ms uint32) {
	if ms != 2000 {
		panic("unexpected delay")
	}
	p.pauses++
}
func (*simulationPlatform) SensorError() { panic("unexpected sensor error") }

func TestSimulationStartsAnotherRound(t *testing.T) {
	p := new(simulationPlatform)
	defer func() {
		if recovered := recover(); recovered != "second round started" {
			t.Fatalf("simulation stopped unexpectedly: %v", recovered)
		}
		if p.rounds != 2 || p.pauses != 1 {
			t.Fatalf("rounds=%d pauses=%d", p.rounds, p.pauses)
		}
	}()
	RunSimulation(p)
	t.Fatal("simulation returned")
}

func TestSimulationOpensPathAboveBricks(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.speedTenths = simulationSpeedTenths
	controller := newSimulationController(1, 4)
	var openedPath, reachedTop bool
	for step := uint32(0); step < 100000 && !s.finished; step++ {
		paddle, pressed := controller.input(&s)
		s.step(paddle, 0, pressed, step*frameCycles)
		centerOpen := true
		for row := range rows {
			if s.bricks[row][cols/2] {
				centerOpen = false
				break
			}
		}
		if centerOpen && s.remaining > 0 {
			openedPath = true
		}
		if openedPath && s.ballY+ballSize <= brickTop {
			reachedTop = true
		}
	}
	if !openedPath || !reachedTop {
		t.Fatalf("opened path=%t, reached above bricks=%t", openedPath, reachedTop)
	}
}

func TestSimulationAimVariesBetweenRounds(t *testing.T) {
	previous := 4
	for round := 0; round < 20; round++ {
		controller := newSimulationController(42, previous)
		if controller.aimOffset == previous || controller.aimOffset < -3 || controller.aimOffset > 3 {
			t.Fatalf("round %d: previous=%d offset=%d", round, previous, controller.aimOffset)
		}
		previous = controller.aimOffset
	}
}

func TestSimulationUsesGyroSpeedGesture(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rate     int
		observed func(*simulationPlatform) bool
	}{
		{"increase", -50, func(p *simulationPlatform) bool { return p.sawSpeedIncrease }},
		{"decrease", 50, func(p *simulationPlatform) bool { return p.sawSpeedDecrease }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &simulationPlatform{speedRate: tc.rate}
			defer func() {
				if recovered := recover(); recovered != "second round started" {
					t.Fatalf("simulation stopped unexpectedly: %v", recovered)
				}
				if p.gyroReads == 0 || !tc.observed(p) {
					t.Fatalf("gyro reads=%d, speed change displayed=%t", p.gyroReads, tc.observed(p))
				}
			}()
			RunSimulation(p)
			t.Fatal("simulation returned")
		})
	}
}

func TestSimulationUsesAngledPaddleBounces(t *testing.T) {
	var s gameState
	s.init(1, 0, false)
	s.speedTenths = simulationSpeedTenths
	controller := newSimulationController(1, 4)
	for step := uint32(0); step < 100000 && !s.finished; step++ {
		previousVY := s.vy
		paddle, pressed := controller.input(&s)
		s.step(paddle, 0, pressed, step*frameCycles)
		if controller.reachedTop && previousVY > 0 && s.vy < 0 && abs(s.vx) >= 3.5 {
			return
		}
	}
	t.Fatal("simulation did not produce an angled paddle bounce after reaching the top")
}
