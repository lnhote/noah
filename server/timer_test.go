package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func fireEvent() {
	println("fire Event")
}

func TestTimerEventIsFired(t *testing.T) {
	et := newEventTimer(fireEvent, 1000)
	et.Start()
	time.Sleep(time.Millisecond * 1010)
	println("et.TimeSinceLastFired()", et.TimeSinceLastFired())
	assert.True(t, et.TimeSinceLastFired() < 50)
	time.Sleep(time.Millisecond * 1000)
	println("et.TimeSinceLastFired()", et.TimeSinceLastFired())
	assert.True(t, et.TimeSinceLastFired() < 50)
	et.Stop()
}

func TestTimerStop(t *testing.T) {
	et := newEventTimer(fireEvent, 500)
	et.Start()
	time.Sleep(time.Millisecond * 1010)
	println("call et.Stop")
	et.Stop()
	println("et.TimeSinceLastFired()", et.TimeSinceLastFired())
	lastTime := et.LastFiredTime()
	time.Sleep(time.Millisecond * 1000)
	assert.Equal(t, lastTime, et.LastFiredTime())
}

func TestTimerRestart(t *testing.T) {
	et := newEventTimer(fireEvent, 500)
	et.Start()
	time.Sleep(time.Millisecond * 1010)
	println("call et.Stop")
	et.Stop()
	println("et.TimeSinceLastFired()", et.TimeSinceLastFired())
	lastTime := et.LastFiredTime()
	time.Sleep(time.Millisecond * 1000)
	assert.Equal(t, lastTime, et.LastFiredTime())

	et.Start()
	time.Sleep(time.Millisecond * 1010)
	println("et.TimeSinceLastFired()", et.TimeSinceLastFired())
	assert.True(t, et.TimeSinceLastFired() < 50)
	time.Sleep(time.Millisecond * 1000)
	println("et.TimeSinceLastFired()", et.TimeSinceLastFired())
	assert.True(t, et.TimeSinceLastFired() < 50)
	et.Stop()
}

func TestTimerReset(t *testing.T) {
	et := newEventTimer(fireEvent, 500)
	et.Start()
	time.Sleep(time.Millisecond * 1010)
	lastTime := et.LastFiredTime()
	println("et.lastTime", lastTime)
	for i := 0; i < 3; i++ {
		println("call et.Reset")
		et.Reset()
		time.Sleep(time.Millisecond * 400)
		assert.Equal(t, lastTime, et.LastFiredTime())
	}
}
