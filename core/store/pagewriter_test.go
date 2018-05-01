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
}

func TestPageWriter_saveRecord_basic(t *testing.T) {
	pageSize := 32 * 1024
	newRepo, _ := CreateRepo("test/TestPageWriter_saveRecord", pageSize, int64(pageSize*10))
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
	assert.Equal(t, 1000+8, c)

	c, err = w.saveRecord(recB, LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	n += c
	log.Printf("recB written %d, total writen %d", c, n)
	assert.Equal(t, 97270+3*8, c)

	c, err = w.saveRecord(recC, LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	n += c
	log.Printf("recC written %d, total writen %d", c, n)
	assert.Equal(t, 8000+8, c)
	assert.True(t, n < 32*1024*4)
}

func TestMustWriteRecordToFile(t *testing.T) {
	pageSize := 128
	newRepo, _ := CreateRepo("test/testMustWriteRecordToFile", pageSize, int64(pageSize*1))
	w := newRepo.Writer
	rec, err := NewRecord(getRandRecord(10), LogEntry, FullType, w.crc)
	n := w.mustWriteRecordToFile(rec)
	assert.Nil(t, err)
	assert.Equal(t, 10+8, n)
	recBytes := rec.MustMarshal()
	log.Print(newRepo.walFileName)
	log.Print("rec", rec, "\nrecBytes", recBytes)
	bytes, err := ioutil.ReadFile(newRepo.walFileName)
	assert.Equal(t, recBytes[:18], bytes[:18])
	log.Print("ReadFramesFromBytes", bytes[:18])
	recList, err := ReadFramesFromFile(newRepo.walFileName, pageSize)
	log.Printf("reclist = %+v", recList)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recList))
	assert.Equal(t, uint16(10), recList[0].Size)
	assert.Equal(t, FullType, recList[0].FType)
}

func TestPageWriter_SaveRecord_LogEntry(t *testing.T) {
	pageSize := 128
	newRepo, _ := CreateRepo("test/TestPageWriter_SaveLogEntry", pageSize, int64(pageSize*1))
	w := newRepo.Writer
	data := getRandRecord(10)
	n, err := w.saveRecord(data, LogEntry)
	assert.Nil(t, err)
	assert.Equal(t, 10+8, n)
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
	newRepo, _ := CreateRepo("tmp/TestPageWriter_SaveState", pageSize, int64(pageSize*2))
	w := newRepo.Writer

	testState := &core.PersistentState{10, 5, core.NewLogRepo()}
	assert.Nil(t, w.SaveState(testState))

	entBytes, err := json.Marshal(testState)
	assert.Equal(t, 43, len(entBytes))
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
	assert.Equal(t, uint16(43), recList[0].Size)
	assert.Equal(t, FullType, recList[0].FType)
}

func TestPageWriter_saveRecordCombine(t *testing.T) {
	pageSize := 32 * 1024
	newRepo, _ := CreateRepo("tmp/TestPageWriter_saveRecordCombine", pageSize, int64(pageSize*5))
	w := newRepo.Writer

	var n, err = w.saveRecord(getRandRecord(1000), LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1000+8, n)
	n, err = w.saveRecord(getRandRecord(97270), LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 97270+3*8, n)
	n, err = w.saveRecord(getRandRecord(8000), LogEntry)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 8000+8, n)

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
