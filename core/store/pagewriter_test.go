package store

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"

	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/core/entity"
	"github.com/stretchr/testify/assert"
)

func TestGetFrameType(t *testing.T) {
	assert.Equal(t, FullType, getFrameType(1024, 1016, 0))
	assert.Equal(t, FirstType, getFrameType(1024, 1024, 0))
	assert.Equal(t, FirstType, getFrameType(100, 1024, 0))
	assert.Equal(t, LastType, getFrameType(1024, 1024, 1000))
	assert.Equal(t, MiddleType, getFrameType(1024, 2000, 10))

	assert.Equal(t, FullType, getFrameType(128, 36, 0))  // 128-36-8=84
	assert.Equal(t, FirstType, getFrameType(84, 80, 0))  // 84-76-8=0
	assert.Equal(t, LastType, getFrameType(128, 80, 76)) // 128-4-8=116
	assert.Equal(t, FullType, getFrameType(116, 85, 0))  // 116-85-8=23
}

func TestPageWriter_saveRecord_basic(t *testing.T) {
	pageSize := 32 * 1024
	newRepo, _ := CreateRepo("test/TestPageWriter_saveRecord_basic", pageSize, int64(pageSize*10))
	w := newRepo.Writer
	recA := getRandRecord(1000)
	assert.Equal(t, 1000, len(recA))
	recB := getRandRecord(97270)
	assert.Equal(t, 97270, len(recB))
	recC := getRandRecord(8000)
	assert.Equal(t, 8000, len(recC))
	n := 0
	var c, err = w.saveRecord(recA, LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	n += c
	log.Printf("recA written %d, total writen %d", c, n)
	assert.Equal(t, 1000, c)

	c, err = w.saveRecord(recB, LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	n += c
	log.Printf("recB written %d, total writen %d", c, n)
	assert.Equal(t, 97270, c)

	c, err = w.saveRecord(recC, LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	n += c
	log.Printf("recC written %d, total writen %d", c, n)
	assert.Equal(t, 8000, c)
	assert.True(t, n < 32*1024*4)
}

func TestMustWriteRecordToFile(t *testing.T) {
	pageSize := 128
	newRepo, _ := CreateRepo("test/TestMustWriteRecordToFile", pageSize, int64(pageSize*1))
	w := newRepo.Writer
	rec, err := NewRecord(getRandRecord(10), LogEntry, FullType, w.crc)
	n := w.mustWriteRecordToFile(rec)
	assert.Nil(t, err)
	assert.Equal(t, 10+8, n)
	recBytes := rec.MustMarshal()
	log.Print(newRepo.walFileName)
	log.Print("rec", rec, "\nrecBytes", recBytes)
	bytesFromFile, err := ioutil.ReadFile(newRepo.walFileName)
	assert.Equal(t, recBytes[:18], bytesFromFile[:18])
	log.Print("ReadFramesFromBytes", bytesFromFile[:18])
	recList, err := ReadFramesFromFile(newRepo.walFileName, pageSize)
	log.Printf("reclist = %+v", recList)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recList))
	assert.Equal(t, uint16(10), recList[0].Size)
	assert.Equal(t, FullType, recList[0].FType)
}

func TestPageWriter_SaveRecord_LogEntry(t *testing.T) {
	pageSize := 128
	newRepo, _ := CreateRepo("test/TestPageWriter_SaveRecord_LogEntry", pageSize, int64(pageSize*1))
	w := newRepo.Writer
	data := getRandRecord(10)
	n, err := w.saveRecord(data, LogEntry)
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	rec, err := NewRecord(data, LogEntry, FullType, w.crc)
	recBytes := rec.MustMarshal()
	assert.Equal(t, uint32(0x7e39d314), rec.Crc)
	assert.Equal(t, 2117718804, int(rec.Crc))
	assert.True(t, bytes.Equal([]byte{0x7e, 0x39, 0xd3, 0x14}, recBytes[:4]))

	bytesFromFile, err := ioutil.ReadFile(newRepo.walFileName)
	assert.Equal(t, recBytes[:18], bytesFromFile[:18])
	// check saveRecord.crc
	assert.Equal(t, []byte{0x7e, 0x39, 0xd3, 0x14}, bytesFromFile[:4])
	recList, err := ReadFramesFromFile(newRepo.walFileName, pageSize)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recList))
	assert.Equal(t, uint16(10), recList[0].Size)
	assert.Equal(t, FullType, recList[0].FType)
	assert.Equal(t, uint32(0x7e39d314), recList[0].Crc)
}

func TestPageWriter_SaveLogEntry(t *testing.T) {
	pageSize := 128
	newRepo, _ := CreateRepo("test/TestPageWriter_SaveLogEntry", pageSize, int64(pageSize*1))
	w := newRepo.Writer
	ent := &core.LogEntry{
		Command: &entity.Command{CommandType: entity.CmdSet, Key: "name", Value: []byte("hunter")},
		Index:   1,
		Term:    1,
	}
	assert.Nil(t, w.SaveLogEntry(ent))

	entBytes, err := json.Marshal(ent)
	assert.Equal(t, 80, len(entBytes))
	rec, err := NewRecord(entBytes, LogEntry, FullType, w.crc)
	assert.Nil(t, err)
	recBytes := rec.MustMarshal()
	log.Printf("rec %+v\nrec bytes: %d\n", rec, recBytes)

	fileBytes, err := ioutil.ReadFile(newRepo.walFileName)
	assert.Nil(t, err)
	log.Print("ReadFramesFromFile", fileBytes[:88])
	assert.Equal(t, recBytes[:18], fileBytes[:18])

	recList, err := ReadFramesFromFile(newRepo.walFileName, pageSize)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recList))
	assert.Equal(t, uint16(80), recList[0].Size)
	assert.Equal(t, FullType, recList[0].FType)
}

func TestPageWriter_SaveState(t *testing.T) {
	pageSize := 128
	newRepo, _ := CreateRepo("test/TestPageWriter_SaveState", pageSize, int64(pageSize*2))
	w := newRepo.Writer

	testState := &core.PersistentState{10, 5, core.NewLogRepo()}
	assert.Nil(t, w.SaveState(testState))

	entBytes, err := json.Marshal(testState)
	assert.Equal(t, 36, len(entBytes))
	rec, err := NewRecord(entBytes, State, FullType, w.crc)
	assert.Nil(t, err)
	recBytes := rec.MustMarshal()
	log.Printf("rec %+v\nrec bytes: %d\n", rec, recBytes)

	fileBytes, err := ioutil.ReadFile(newRepo.walFileName)
	assert.Nil(t, err)
	log.Print("ReadFramesFromFile", fileBytes[:51])
	assert.Equal(t, recBytes[:18], fileBytes[:18])

	recList, err := ReadFramesFromFile(newRepo.walFileName, pageSize)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recList))
	assert.Equal(t, uint16(36), recList[0].Size)
	assert.Equal(t, FullType, recList[0].FType)
}

func TestPageWriter_saveRecordCombine(t *testing.T) {
	pageSize := 32 * 1024
	newRepo, _ := CreateRepo("test/TestPageWriter_saveRecordCombine", pageSize, int64(pageSize*5))
	w := newRepo.Writer

	var n, err = w.saveRecord(getRandRecord(1000), LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1000, n)
	n, err = w.saveRecord(getRandRecord(97270), LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 97270, n)
	n, err = w.saveRecord(getRandRecord(8000), LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 8000, n)

	recList, err := ReadFramesFromFile(newRepo.walFileName, pageSize)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(recList))

	// 32768
	assert.Equal(t, uint16(1000), recList[0].Size)
	assert.Equal(t, FullType, recList[0].FType)
	assert.Equal(t, uint16(31752), recList[1].Size)
	assert.Equal(t, FirstType, recList[1].FType)

	assert.Equal(t, uint16(32760), recList[2].Size)
	assert.Equal(t, MiddleType, recList[2].FType)

	assert.Equal(t, uint16(32758), recList[3].Size)
	assert.Equal(t, LastType, recList[3].FType)

	assert.Equal(t, uint16(8000), recList[4].Size)
	assert.Equal(t, FullType, recList[4].FType)
}

func TestRestoreLogEntriesAndState(t *testing.T) {
	pageSize := 128
	newRepo, _ := CreateRepo("test/TestRestoreLogEntriesAndState", pageSize, int64(pageSize*3))
	w := newRepo.Writer
	ents := []*core.LogEntry{{
		Command: &entity.Command{CommandType: entity.CmdSet, Key: "name", Value: []byte("hunter")},
		Index:   1,
		Term:    1,
	}, {
		Command: &entity.Command{CommandType: entity.CmdSet, Key: "name1", Value: []byte("hunter1")},
		Index:   2,
		Term:    1,
	}}
	testState := core.NewPersistentState(10, 5, []*core.LogEntry{})
	assert.Nil(t, w.SaveState(testState))
	for _, ent := range ents {
		assert.Nil(t, w.SaveLogEntry(ent))
	}
	state, err := RestoreLogEntriesAndState(newRepo.walFileName, pageSize)
	assert.Nil(t, err)
	assert.Equal(t, 10, state.Term)
	assert.Equal(t, 5, state.LastVotedServerId)
	logEntires := state.Logs.GetAllLogs()
	assert.Equal(t, 3, len(logEntires))
}
