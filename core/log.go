package core

import (
	"fmt"
	"github.com/lnhote/noah/core/entity"
)

// LogEntry includes the log index, term and content
type LogEntry struct {
	Command *entity.Command
	Index   int
	Term    int
}

// LogRepo
type LogRepo struct {
	// logs: logindex => log struct, start from 1
	logs      map[int]*LogEntry
	lastIndex int
}

// NewLogRepo
func NewLogRepo() *LogRepo {
	return &LogRepo{logs: map[int]*LogEntry{}, lastIndex: 0}
}

// NewLogRepoWithLogs
func NewLogRepoWithLogs(ents []*LogEntry) *LogRepo {
	repo := &LogRepo{logs: map[int]*LogEntry{}, lastIndex: 0}
	for _, ent := range ents {
		repo.SaveLogEntry(ent)
	}
	return repo
}

// GetLastIndex
func (l *LogRepo) GetLastIndex() int {
	return l.lastIndex
}

// GetNextIndex
func (l *LogRepo) GetNextIndex() int {
	return l.GetLastIndex() + 1
}

// GetLastTerm
func (l *LogRepo) GetLastTerm() int {
	if log, err := l.GetLogEntry(l.lastIndex); err == nil {
		return log.Term
	}
	return 0
}

// GetLogTerm
func (l *LogRepo) GetLogTerm(index int) (int, error) {
	entry, err := l.GetLogEntry(index)
	if err != nil {
		return 0, err
	}
	return entry.Term, nil
}

// GetLogList
func (l *LogRepo) GetLogList(start int) []*LogEntry {
	newLogs := []*LogEntry{}
	for i := start; i <= l.GetLastIndex(); i++ {
		newLogs = append(newLogs, l.logs[i])
	}
	return newLogs
}

// GetAllLogs
func (l *LogRepo) GetAllLogs() []*LogEntry {
	return l.GetLogList(0)
}

// GetLogEntry
func (l *LogRepo) GetLogEntry(index int) (*LogEntry, error) {
	if entry, ok := l.logs[index]; ok {
		return entry, nil
	}
	return nil, fmt.Errorf("InvalidLogIndex(%d)", index)
}

// SaveLogEntry
func (l *LogRepo) SaveLogEntry(log *LogEntry) {
	l.logs[log.Index] = log
	if log.Index > l.lastIndex {
		l.lastIndex = log.Index
	}
}

// Delete
func (l *LogRepo) Delete(index int) {
	delete(l.logs, index)
}
