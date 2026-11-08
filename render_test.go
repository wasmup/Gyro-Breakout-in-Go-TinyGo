package game

import "testing"

type recordedRect struct {
	x, y, w, h int
	color      uint16
}
type testPlatform struct{ rects []recordedRect }

func (*testPlatform) Cycles() uint32              { return 0 }
func (*testPlatform) Seed() uint32                { return 0 }
func (*testPlatform) ButtonPressed() bool         { return false }
func (*testPlatform) GyroRates() (int, int, bool) { return 0, 0, true }
func (p *testPlatform) Rect(x, y, w, h int, c uint16) {
	p.rects = append(p.rects, recordedRect{x, y, w, h, c})
}
func (*testPlatform) Delay(uint32) {}
func (*testPlatform) SensorError() {}
func TestRestoreSurvivingBonusBrick(t *testing.T) {
	p := new(testPlatform)
	r := renderer{p}
	var s gameState
	s.bricks[0][0] = true
	s.bonusBrick = 0
	r.restore(&s, 1, brickTop)
	if len(p.rects) != 1 || p.rects[0] != (recordedRect{1, brickTop, brickWidth, brickHeight, blue}) {
		t.Fatalf("unexpected rectangles: %#v", p.rects)
	}
	p.rects = nil
	s.bricks[0][0] = false
	r.restore(&s, 1, brickTop)
	if len(p.rects) != 0 {
		t.Fatal("restored destroyed brick")
	}
}
func TestEraseBallOnlyExposedArea(t *testing.T) {
	p := new(testPlatform)
	renderer{p}.eraseBall(10, 20, 12, 20)
	if len(p.rects) != 1 || p.rects[0] != (recordedRect{10, 20, 2, 10, black}) {
		t.Fatalf("unexpected rectangles: %#v", p.rects)
	}
}

func TestCalibrationMessage(t *testing.T) {
	p := new(testPlatform)
	ShowCalibration(p)
	if len(p.rects) == 0 || p.rects[0] != (recordedRect{0, 0, screenW, screenH, black}) {
		t.Fatal("calibration screen was not cleared")
	}
	var firstLine, secondLine bool
	for _, rect := range p.rects[1:] {
		if rect.color == white && rect.y >= 100 && rect.y < 114 {
			firstLine = true
		}
		if rect.color == white && rect.y >= 124 && rect.y < 138 {
			secondLine = true
		}
	}
	if !firstLine || !secondLine {
		t.Fatal("calibration message is missing a line")
	}
}

func TestDrawFrameUpdatesDestroyedBrickAndPaddle(t *testing.T) {
	p := new(testPlatform)
	var old gameState
	old.init(1, 0, false)
	state := old
	state.bricks[0][0] = false
	state.paddle += 3
	displayed := renderer{p}.drawFrame(&old, &state, state.speedTenths)
	if displayed != state.speedTenths {
		t.Fatalf("displayed speed = %d", displayed)
	}
	var erasedBrick, drewPaddle bool
	for _, rect := range p.rects {
		if rect == (recordedRect{1, brickTop, brickWidth, brickHeight, black}) {
			erasedBrick = true
		}
		if rect == (recordedRect{state.paddle, paddleY, paddleW, 7, white}) {
			drewPaddle = true
		}
	}
	if !erasedBrick || !drewPaddle {
		t.Fatal("frame did not update destroyed brick and paddle")
	}
}

func TestEndMessages(t *testing.T) {
	for _, tc := range []struct {
		name      string
		win       bool
		text      string
		scale     int
		color     uint16
		firstRows []uint8
	}{
		{"win", true, "YOU WIN", 3, green, []uint8{17, 14, 17, 0, 17, 14, 17}},
		{"loss", false, "MAYBE NEXT TIME", 2, red, []uint8{17, 14, 17, 30, 31, 0, 17, 31, 17, 31, 0, 31, 14, 17, 31}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := new(testPlatform)
			renderer{p}.message(tc.win)
			if len(p.rects) == 0 || p.rects[0] != (recordedRect{0, 109, screenW, 21, black}) {
				t.Fatal("message background was not cleared")
			}
			pixels := make(map[recordedRect]bool)
			for _, rect := range p.rects[1:] {
				pixels[rect] = true
			}
			left := (screenW - (len(tc.text)*6-1)*tc.scale) / 2
			for i, row := range tc.firstRows {
				for x := range 5 {
					rect := recordedRect{left + i*6*tc.scale + x*tc.scale, 109, tc.scale, tc.scale, tc.color}
					if got, want := pixels[rect], row&(1<<uint(4-x)) != 0; got != want {
						t.Fatalf("character %d pixel %d: got %t want %t", i, x, got, want)
					}
				}
			}
		})
	}
}

func TestElapsedTimeFormatting(t *testing.T) {
	for _, tc := range []struct {
		seconds uint64
		want    string
	}{
		{0, "TIME 00:00"}, {65, "TIME 01:05"}, {5999, "TIME 99:59"}, {6000, "TIME 100:00"},
	} {
		if got := formatElapsed(tc.seconds); got != tc.want {
			t.Errorf("%d: got %q, want %q", tc.seconds, got, tc.want)
		}
	}
	p := new(testPlatform)
	renderer{p}.elapsedTime(65)
	if len(p.rects) < 2 || p.rects[0] != (recordedRect{0, 140, screenW, 14, black}) {
		t.Fatal("time was not drawn on a cleared background")
	}
}

func TestDrawDamagedBrickRed(t *testing.T) {
	p := new(testPlatform)
	var old gameState
	old.init(1, 0, false)
	state := old
	state.damaged[0][0] = true
	renderer{p}.drawFrame(&old, &state, state.speedTenths)
	want := recordedRect{1, brickTop, brickWidth, brickHeight, red}
	for _, rect := range p.rects {
		if rect == want {
			return
		}
	}
	t.Fatal("damaged brick was not redrawn in red")
}
