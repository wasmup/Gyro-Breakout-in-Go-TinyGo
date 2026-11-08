package main

// gyroFilter removes the stationary bias and averages the last two samples.
// Sensor sensitivity at the configured range is 8.75 millidegrees per second.
type gyroFilter struct {
	offsets [3]int32
	totals  [3]int32
	history [3][2]int32
	index   int
	count   int
}

func (f *gyroFilter) rates(axes [3]int16) [3]int {
	if f.count < len(f.history[0]) {
		f.count++
	}
	var rates [3]int
	for i := range axes {
		f.totals[i] -= f.history[i][f.index]
		f.history[i][f.index] = int32(axes[i]) - f.offsets[i]
		f.totals[i] += f.history[i][f.index]
		rate := int(f.totals[i] / int32(f.count) * 875 / 100000)
		if rate < -250 {
			rate = -250
		} else if rate > 250 {
			rate = 250
		}
		rates[i] = rate
	}
	f.index = (f.index + 1) % len(f.history[0])
	return rates
}
