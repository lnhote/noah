package store

import (
	"encoding/json"
	"hash"
	"io/ioutil"
	"log"

	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/core/errmsg"
)

//Record frame format:
//+---------+-----------+-----------+-----------+--- ... ---+
//|CRC (4B) | Size (2B) | Type (1B) | DType(1B) | Payload   |
//+---------+-----------+-----------+-----------+--- ... ---+
type Record struct {
	Crc   uint32
	Size  uint16
	FType frameType
	DType dataType
	Data  []byte
}

const (
	FrameHeadSize = 8
)

type frameType uint8

const (
	ZeroType frameType = iota
	FullType
	FirstType
	LastType
	MiddleType
)

type dataType uint8

const (
	LogEntry dataType = iota
	State
	CRC
)

func (r *Record) Marshal() ([]byte, error) {
	frameHead := make([]byte, FrameHeadSize)
	frameHead[3] = byte(r.Crc & 0x000000ff)
	frameHead[2] = byte(r.Crc >> 8 & 0x000000ff)
	frameHead[1] = byte(r.Crc >> 16 & 0x000000ff)
	frameHead[0] = byte(r.Crc >> 24 & 0x000000ff)

	frameHead[5] = byte(r.Size & 0x000000ff)
	frameHead[4] = byte(r.Size >> 8 & 0x000000ff)

	frameHead[6] = byte(r.FType & 0x000000ff)

	frameHead[7] = byte(r.DType & 0x000000ff)
	frame := make([]byte, 0, FrameHeadSize+len(r.Data))
	frame = append(frame, frameHead...)
	frame = append(frame, r.Data...)
	return frame, nil
}
func isEmptyFrame(data []byte) bool {
	if len(data) <= FrameHeadSize {
		return true
	}
	for _, b := range data[0:FrameHeadSize] {
		if b != 0x0 {
			return false
		}
	}
	return true
}

func (r *Record) Unmarshal(data []byte) error {
	if len(data) < FrameHeadSize {
		return errmsg.DataTooShort
	}
	if isEmptyFrame(data) {
		return errmsg.EOF
	}
	r.Crc = uint32(data[0])<<24&0xff000000 +
		uint32(data[1])<<16&0x00ff0000 +
		uint32(data[2])<<8&0x0000ff00 +
		uint32(data[3])&0x000000ff
	r.Size = uint16(data[4])<<8&0xff00 + uint16(data[5])&0x00ff
	if int(r.Size) > len(data)-FrameHeadSize {
		log.Printf("DataTooShort: r.Size = %d, data lenth = %d", r.Size, len(data))
		return errmsg.DataTooShort
	}
	r.FType = frameType(data[6])
	r.DType = dataType(data[7])
	r.Data = data[FrameHeadSize : FrameHeadSize+r.Size]
	return nil
}

func (r *Record) MustMarshal() []byte {
	bytes, err := r.Marshal()
	if err != nil {
		panic(errmsg.EncodingError)
	}
	return bytes
}

func (r *Record) FrameSize() int {
	return FrameHeadSize + int(r.Size)
}

func NewRecord(bytes []byte, dtype dataType, ftype frameType, crc hash.Hash32) (*Record, error) {
	rec := &Record{Data: bytes, Size: uint16(len(bytes)), DType: dtype, FType: ftype}
	crc.Reset()
	var n, err = crc.Write(bytes)
	if err != nil {
		return nil, err
	}
	if n != len(bytes) {
		return nil, errmsg.CRCShortWrite
	}
	rec.Crc = crc.Sum32()
	return rec, nil
}

func RestoreLogEntriesAndState(filename string, pageSize int) (*core.PersistentState, error) {
	logEntries := make([]*core.LogEntry, 0, 1)
	records, err := ReadFramesFromFile(filename, pageSize)
	if err != nil {
		return nil, err
	}
	var logEntry *core.LogEntry
	var state *core.PersistentState
	logEntryBuf := make([]byte, 0, 1)
	stateBuf := make([]byte, 0, 1)
	for i := 0; i < len(records); i++ {
		record := records[i]
		if record.DType == LogEntry {
			switch record.FType {
			case FullType:
				logEntry = &core.LogEntry{}
				if err := json.Unmarshal(record.Data, logEntry); err != nil {
					panic(err)
				}
				logEntries = append(logEntries, logEntry)
			case FirstType:
				logEntryBuf = append(logEntryBuf, record.Data...)
			case MiddleType:
				logEntryBuf = append(logEntryBuf, record.Data...)
			case LastType:
				logEntryBuf = append(logEntryBuf, record.Data...)
				logEntry = &core.LogEntry{}
				if err := json.Unmarshal(logEntryBuf, logEntry); err != nil {
					panic(err)
				}
				logEntries = append(logEntries, logEntry)
				logEntryBuf = make([]byte, 0, 1)
			default:
			}
		} else if record.DType == State {
			switch record.FType {
			case FullType:
				state = &core.PersistentState{}
				if err := json.Unmarshal(record.Data, state); err != nil {
					panic(err)
				}
			case FirstType:
				stateBuf = append(stateBuf, record.Data...)
			case MiddleType:
				stateBuf = append(stateBuf, record.Data...)
			case LastType:
				stateBuf = append(stateBuf, record.Data...)
				state = &core.PersistentState{}
				if err := json.Unmarshal(stateBuf, state); err != nil {
					panic(err)
				}
				stateBuf = make([]byte, 0, 1)
			default:
			}
		}
	}
	if len(logEntries) > 0 {
		state.Logs = core.NewLogRepoWithLogs(logEntries)
	} else {
		state.Logs = core.NewLogRepo()
	}
	return state, nil
}

func ReadFramesFromFile(filename string, pageSize int) ([]*Record, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ReadFramesFromBytes(bytes, pageSize)
}

func ReadFramesFromBytes(bytes []byte, pageSize int) ([]*Record, error) {
	records := []*Record{}
	totalPage := len(bytes) / pageSize
	if len(bytes)%pageSize != 0 {
		totalPage += 1
	}
	for page := 0; page < totalPage; page++ {
		offset := page * pageSize
		recordsPerPage, err := ReadRecordsByPage(bytes[offset:], pageSize)
		if err != nil && err != errmsg.EOF {
			return nil, err
		}
		if err == errmsg.EOF {
			break
		}
		records = append(records, recordsPerPage...)
	}
	return records, nil
}

func ReadRecordsByPage(bytes []byte, pageSize int) ([]*Record, error) {
	records := []*Record{}
	var bytesInPage []byte
	// total bytes is the min of [0, len-1] and [0, pageSize-1]
	if len(bytes) > pageSize {
		bytesInPage = bytes[:pageSize]
	} else {
		bytesInPage = bytes
	}
	for offset := 0; offset < len(bytesInPage); {
		if len(bytesInPage)-offset <= FrameHeadSize {
			break
		}
		newRecord := &Record{}
		err := newRecord.Unmarshal(bytesInPage[offset:])
		if err != nil && err != errmsg.EOF {
			return nil, err
		}
		if err == errmsg.EOF {
			break
		}
		records = append(records, newRecord)
		offset += newRecord.FrameSize()
	}
	return records, nil
}
