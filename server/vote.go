package server

import (
	"fmt"
	"sync"
)

type voteCounter struct {
	Accept     int
	Reject     int
	Fail       int
	Total      int
	Timeout    int
	countMutex *sync.Mutex
	WG         *sync.WaitGroup
}

func newVoteCounter(total int) *voteCounter {
	return &voteCounter{
		Total:      total,
		countMutex: &sync.Mutex{},
		WG:         &sync.WaitGroup{},
	}
}

func (v *voteCounter) String() string {
	return fmt.Sprintf("voteCounter:{accept:%d, reject:%d, fail:%d, total:%d, timeout:%d}",
		v.Accept, v.Reject, v.Fail, v.Total, v.Timeout)
}

func (v *voteCounter) Reset() {
	v.Accept = 0
	v.Reject = 0
	v.Fail = 0
	v.Timeout = 0
}

func (v *voteCounter) IsEnough() bool {
	return v.Accept*2 > v.Total
}

func (v *voteCounter) AddAccept() {
	v.countMutex.Lock()
	v.Accept = v.Accept + 1
	v.countMutex.Unlock()
}

func (v *voteCounter) AddReject() {
	v.countMutex.Lock()
	v.Reject = v.Reject + 1
	v.countMutex.Unlock()
}

func (v *voteCounter) AddTimeout() {
	v.countMutex.Lock()
	v.Timeout = v.Timeout + 1
	v.countMutex.Unlock()
}

func (v *voteCounter) AddFail() {
	v.countMutex.Lock()
	v.Fail = v.Fail + 1
	v.countMutex.Unlock()
}
