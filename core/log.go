package core

import (
	"fmt"
	"github.com/lnhote/noah/core/entity"
)

type LogEntry struct {
	Command *entity.Command
	Index   int
	Term    int
}

type LogRepo struct {
	// logs: logindex => log struct, start from 1
	logs      map[int]*LogEntry
	lastIndex int
}

func NewLogRepo() *LogRepo {
	return &LogRepo{logs: map[int]*LogEntry{}, lastIndex: 0}
}

func (l *LogRepo) GetLastIndex() int {
	return l.lastIndex
}

func (l *LogRepo) GetNextIndex() int {
	return l.GetLastIndex() + 1
}

func (l *LogRepo) GetLastTerm() int {
	if log, err := l.GetLogEntry(l.lastIndex); err == nil {
		return log.Term
	} else {
		return 0
	}
}

func (l *LogRepo) GetLogTerm(index int) (int, error) {
	entry, err := l.GetLogEntry(index)
	if err != nil {
		return 0, err
	}
	return entry.Term, nil
}

func (l *LogRepo) GetLogList(start int) []*LogEntry {
	newLogs := []*LogEntry{}
	for i := start; i <= l.GetLastIndex(); i++ {
		newLogs = append(newLogs, l.logs[i])
	}
	return newLogs
}

func (ll *LogRepo) GetLogEntry(index int) (*LogEntry, error) {
	if entry, ok := ll.logs[index]; ok {
		return entry, nil
	} else {
		return nil, fmt.Errorf("InvalidLogIndex(%d)", index)
	}
}

func (ll *LogRepo) SaveLogEntry(log *LogEntry) {
	ll.logs[log.Index] = log
	if log.Index > ll.lastIndex {
		ll.lastIndex = log.Index
	}
}

func (ll *LogRepo) Delete(index int) {
	delete(ll.logs, index)
}
