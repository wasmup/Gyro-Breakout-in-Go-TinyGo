package main

import "testing"

func TestGyroFilterBiasAndMovingAverage(t *testing.T) {
	f := gyroFilter{offsets: [3]int32{100, -100, 0}}
	if got := f.rates([3]int16{10100, -10100, 0}); got != [3]int{87, -87, 0} {
		t.Fatalf("first sample: %v", got)
	}
	if got := f.rates([3]int16{100, -100, 0}); got != [3]int{43, -43, 0} {
		t.Fatalf("two-sample average: %v", got)
	}
	if got := f.rates([3]int16{100, -100, 0}); got != [3]int{} {
		t.Fatalf("old sample was not removed: %v", got)
	}
}

func TestGyroFilterClampsRates(t *testing.T) {
	var f gyroFilter
	if got := f.rates([3]int16{32767, -32768, 0}); got != [3]int{250, -250, 0} {
		t.Fatalf("clamped rates: %v", got)
	}
}
