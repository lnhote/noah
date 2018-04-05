package server

import "time"

type TimerEvent func()

type eventTimer struct {
	DurationInMs int

	lastFiredTime time.Time
	timer         *time.Timer
	stopSigal     chan int
	fireEvent     TimerEvent
}

func NewEventTimer(fireEvent TimerEvent, dur int) *eventTimer {
	et := &eventTimer{
		DurationInMs:  dur,
		fireEvent:     fireEvent,
		stopSigal:     make(chan int, 1),
		lastFiredTime: time.Now(),
	}
	if dur > 0 {
		et.timer = time.NewTimer(time.Duration(et.DurationInMs) * time.Millisecond)
	}
	return et
}

// TimeSinceLastFired returns the time since last fire event in ms.
func (t *eventTimer) TimeSinceLastFired() int64 {
	return int64(time.Now().Sub(t.lastFiredTime)) / 1e6
}

func (t *eventTimer) LastFiredTime() string {
	return t.lastFiredTime.Format("2006-01-02T15:04:05.999")
}

func (t *eventTimer) Start() {
	if t.DurationInMs == 0 {
		return
	}

	go func() {
	timeLoop:
		for {
			select {
			case <-t.timer.C:
				t.fireEvent()
				t.lastFiredTime = time.Now()
				t.Reset()
			case <-t.stopSigal:
				break timeLoop
			}
		}
	}()
}

func (t *eventTimer) Stop() {
	t.stopSigal <- 1
}

// resetTimer reset the timer
func (t *eventTimer) Reset() {
	if t.DurationInMs == 0 || t.timer == nil {
		return
	}
	t.timer.Stop()
	t.timer.Reset(time.Duration(t.DurationInMs) * time.Millisecond)
}
