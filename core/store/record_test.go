package store

import (
	"io/ioutil"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getRandRecord(size int) []byte {
	content := make([]byte, size)
	rand.Read(content)
	return content
}

func TestShift(t *testing.T) {
	frameHead := []byte{0x1A, 0xAA, 0xAA, 0xAA, 0x03, 0xE8, 0x01, 0x00}
	var crc uint32
	crc = uint32(frameHead[0]) << 24 & 0xff000000
	assert.Equal(t, uint32(0x1A000000), crc)
	crc += uint32(frameHead[1]) << 16 & 0x00ff0000
	assert.Equal(t, uint32(0x1AAA0000), crc)
	crc += uint32(frameHead[2]) << 8 & 0x0000ff00
	assert.Equal(t, uint32(0x1AAAAA00), crc)
	crc += uint32(frameHead[3]) & 0x000000ff
	assert.Equal(t, uint32(0x1AAAAAAA), crc)
}

func TestRecord_MustMarshal(t *testing.T) {
	recAData := getRandRecord(1000)
	recA := &Record{Crc: 0x1AAAAAAA, Size: 1000, FType: FullType, DType: LogEntry, Data: recAData}
	encodedRecA := recA.MustMarshal()
	assert.Equal(t, []byte{0x1A, 0xAA, 0xAA, 0xAA, 0x03, 0xE8, 0x01, 0x00}, encodedRecA[0:8])
	assert.Equal(t, 1000, len(encodedRecA[8:]))
	assert.Equal(t, recAData, encodedRecA[8:])
	assert.Equal(t, 1008, len(encodedRecA))
}

func TestRecord_UnmarshalMarshal(t *testing.T) {
	recAData := getRandRecord(1000)
	recA := &Record{Crc: 0x1AAAAAAA, Size: 1000, FType: FullType, DType: LogEntry, Data: recAData}
	encodedRecA := recA.MustMarshal()

	assert.Equal(t, []byte{0x1A, 0xAA, 0xAA, 0xAA, 0x03, 0xE8, 0x01, 0x00}, encodedRecA[0:8])
	assert.Equal(t, 1000, len(encodedRecA[8:]))
	assert.Equal(t, recAData, encodedRecA[8:])
	assert.Equal(t, 1008, len(encodedRecA))
	decodedRecA := &Record{}
	assert.Nil(t, decodedRecA.Unmarshal(encodedRecA))
	assert.Equal(t, recA.Crc, decodedRecA.Crc)
	assert.Equal(t, recA.Size, decodedRecA.Size)
	assert.Equal(t, recA.DType, decodedRecA.DType)
	assert.Equal(t, recA.FType, decodedRecA.FType)
	assert.Equal(t, recA.Data, decodedRecA.Data)
}

func TestReadFromBytes(t *testing.T) {
	// recA 1016, recB 504, recC 1008

	// 1016+8|504+8|504+8|504+8
	pageSize := 1024
	totalBytes := []byte{}
	recA := &Record{Crc: 0x1AAAAAAA, Size: 1016, FType: FullType, DType: LogEntry, Data: getRandRecord(1016)}
	encodedRecA := recA.MustMarshal()
	totalBytes = append(totalBytes, encodedRecA...)

	recB := &Record{Crc: 0x1AAAAAAA, Size: 504, FType: FullType, DType: LogEntry, Data: getRandRecord(504)}
	encodedRecB := recB.MustMarshal()
	totalBytes = append(totalBytes, encodedRecB...)

	recC1 := &Record{Crc: 0x1AAAAAAA, Size: 504, FType: FirstType, DType: LogEntry, Data: getRandRecord(504)}
	encodedRecC1 := recC1.MustMarshal()
	totalBytes = append(totalBytes, encodedRecC1...)

	recC2 := &Record{Crc: 0x1AAAAAAA, Size: 504, FType: LastType, DType: LogEntry, Data: getRandRecord(504)}
	encodedRecC2 := recC2.MustMarshal()
	totalBytes = append(totalBytes, encodedRecC2...)

	recList, err := ReadFramesFromBytes(totalBytes, pageSize)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(recList))
	assert.Equal(t, uint16(1016), recList[0].Size)
	assert.Equal(t, FullType, recList[0].FType)
	assert.Equal(t, uint16(504), recList[1].Size)
	assert.Equal(t, FullType, recList[1].FType)
	assert.Equal(t, uint16(504), recList[2].Size)
	assert.Equal(t, FirstType, recList[2].FType)
	assert.Equal(t, uint16(504), recList[3].Size)
	assert.Equal(t, LastType, recList[3].FType)
}

func TestReadFromBytes_withPad(t *testing.T) {
	// recA 116, recB 56, recC 112

	// 116+8+pad(4)|56+8|56+8|56+8
	pageSize := 128
	totalBytes := []byte{}
	recA := &Record{Crc: 0x1AAAAAAA, Size: 116, FType: FullType, DType: LogEntry, Data: getRandRecord(116)}
	encodedRecA := recA.MustMarshal()
	totalBytes = append(totalBytes, encodedRecA...)
	totalBytes = append(totalBytes, getRandRecord(4)...)

	recB := &Record{Crc: 0x1AAAAAAA, Size: 56, FType: FullType, DType: LogEntry, Data: getRandRecord(56)}
	encodedRecB := recB.MustMarshal()
	totalBytes = append(totalBytes, encodedRecB...)

	recC1 := &Record{Crc: 0x1AAAAAAA, Size: 56, FType: FirstType, DType: LogEntry, Data: getRandRecord(56)}
	encodedRecC1 := recC1.MustMarshal()
	totalBytes = append(totalBytes, encodedRecC1...)

	recC2 := &Record{Crc: 0x1AAAAAAA, Size: 56, FType: LastType, DType: LogEntry, Data: getRandRecord(56)}
	encodedRecC2 := recC2.MustMarshal()
	totalBytes = append(totalBytes, encodedRecC2...)

	testWalFileName := "tmp/TestReadFromBytes_withPad"
	err := ioutil.WriteFile(testWalFileName, totalBytes, 0644)
	assert.Nil(t, err)

	recList, err := ReadFramesFromBytes(totalBytes, pageSize)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(recList))
	assert.Equal(t, uint16(116), recList[0].Size)
	assert.Equal(t, FullType, recList[0].FType)
	assert.Equal(t, uint16(56), recList[1].Size)
	assert.Equal(t, FullType, recList[1].FType)
	assert.Equal(t, uint16(56), recList[2].Size)
	assert.Equal(t, FirstType, recList[2].FType)
	assert.Equal(t, uint16(56), recList[3].Size)
	assert.Equal(t, LastType, recList[3].FType)

	recListFromFile, err := ReadFramesFromFile(testWalFileName, pageSize)
	assert.NotEmpty(t, recListFromFile)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(recListFromFile))
	assert.Equal(t, uint16(116), recListFromFile[0].Size)
	assert.Equal(t, FullType, recListFromFile[0].FType)
	assert.Equal(t, uint16(56), recListFromFile[1].Size)
	assert.Equal(t, FullType, recListFromFile[1].FType)
	assert.Equal(t, uint16(56), recListFromFile[2].Size)
	assert.Equal(t, FirstType, recListFromFile[2].FType)
	assert.Equal(t, uint16(56), recListFromFile[3].Size)
	assert.Equal(t, LastType, recListFromFile[3].FType)
}
