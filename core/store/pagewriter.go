package store

import (
	"encoding/json"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/core/errmsg"
	"github.com/lnhote/noah/core/store/crc"
	"github.com/lnhote/noah/core/store/fs"
)

/**
File layout:
      +-----+-------------+--+----+----------+------+-- ... ----+
File  | r0  |        r1   |P | r2 |    r3    |  r4  |           |
      +-----+-------------+--+----+----------+------+-- ... ----+
      <----- PageSize ------>|<---- PageSize ------>|

rn = variable size records
P = Padding


Record frame format:
+---------+-----------+-----------+-----------+--- ... ---+
|CRC (4B) | Size (2B) | Type (1B) | DType(1B) | Payload   |
+---------+-----------+-----------+-----------+--- ... ---+

CRC = 32bit hash computed over the payload using CRC
Size = Length of the payload data
Type = Type of record
       (ZeroType, FullType, FirstType, LastType, MiddleType )
       The type is used to group a bunch of records together to represent
       blocks that are larger than kBlockSize
Payload = Byte stream as long as specified by the payload size


Here are the cases:

If record is smaller than PageSize:
      +--------------------+--+
File  | record             |P |
      |--------------------|  |
      | CRC+Size...Payload |  | FullType
      +--------------------+--+
      |<---- PageSize ------->|

If record is equal to PageSize:
      +-----------------------+
File  | record                |
      |-----------------------|
      | CRC+Size...Payload    | FullType
      +-----------------------+
      |<---- PageSize ------->|

If record is larger than PageSize:
      +-----------------------+
File  | record                |
      |-----------------------|
      | CRC+Size+First+Payload|
      +-----------------------+
      |<---- PageSize ------->|

A will be stored as a FULL record in the first block.

B will be split into three fragments:
first fragment occupies the restof the first block,
second fragment occupies the entirety of the second block,
and the third fragment occupies a prefix of the third block.
This will leave 2 bytes free in the third block, which will be left empty as the trailer.

C will be stored as a FULL record in the fourth block

Block: 32*1024B=32768B
A: length 1000
B: length 97270
C: length 8000

so the record will be stored like this:

+-------+---------------++-----------------------++---------------------+-++-------------+---------+
|recordA|recordB(First) ||recordB(Middle)        ||recordB(Last)        | ||recordC(Full)|         |
|-+-----|-+-------------||-+---------------------||-+-------------------+-||-+-----------|---------|
|8|1000 |8|31752        ||8|32760                ||8|32758              | ||8|8000       |         |
+-+-----+-+-------------++-+---------------------++-+-------------------+-++-+-----------+---------+
|<---- 32*1024B ------->||<---- 32*1024B ------->||<---- 32*1024B ------->||<---- 32*1024B ------->|

*/

var (
	SegmentSizeBytes int64 = 1024 * 1024 // 1MB
	//SegmentSizeBytes int64 = 64 * 1024 * 1024 // 64MB
	DefaultFlushSize = 128 * 1024 // 128kB
	DefaultPageSize  = 32 * 1024
)

type repo struct {
	dir         string
	walFileName string
	Writer      *pageWriter
}

type pageWriter struct {
	mu     sync.Mutex
	crc    hash.Hash32
	writer *os.File
	buf    []byte

	// offset: the offset of the next read/write in the segment file
	offset int

	// pageSize: how many bytes per page
	pageSize int

	// segmentSize: the size of a segment file
	segmentSize int64
}

func NewPageWriter(f *os.File, prevCrc uint32, pageSize int, segmentSize int64) *pageWriter {
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(errmsg.OpenWALError)
	}
	pw := &pageWriter{
		buf:         make([]byte, DefaultFlushSize),
		offset:      int(offset),
		pageSize:    pageSize,
		segmentSize: segmentSize,
		writer:      f,
		crc:         crc.New(prevCrc, crc32.MakeTable(crc32.Castagnoli)),
	}
	return pw
}

func CreateRepo(dirpath string, pageSize int, segmentSize int64) (*repo, error) {
	if fs.Exist(dirpath) {
		return nil, os.ErrExist
	}
	tmpdirpath := filepath.Clean(dirpath) + ".tmp"
	if fs.Exist(tmpdirpath) {
		if err := os.RemoveAll(tmpdirpath); err != nil {
			return nil, err
		}
	}
	if err := fs.CreateDirAll(tmpdirpath); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%016x-%016x.wal", 0, 0)
	fullFilename := filepath.Join(tmpdirpath, filename)
	f, err := fs.LockFile(fullFilename, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}
	if err = fs.Preallocate(f.File, segmentSize, true); err != nil {
		return nil, err
	}
	return &repo{
		dir:         dirpath,
		walFileName: fullFilename,
		Writer:      NewPageWriter(f.File, 0, pageSize, segmentSize),
	}, nil
}

func CreateDefaultRepo(dirpath string) (*repo, error) {
	return CreateRepo(dirpath, DefaultPageSize, SegmentSizeBytes)
}

// SaveAll persist all stuff to disk
func (pw *pageWriter) SaveAll(ents []*core.LogEntry, state *core.PersistentState) error {
	if len(ents) > 0 {
		for _, ent := range ents {
			if err := pw.SaveLogEntry(ent); err != nil {
				return err
			}
		}
	}
	if err := pw.SaveState(state); err != nil {
		return err
	}
	return pw.sync()
}

func (pw *pageWriter) SaveLogEntry(ent *core.LogEntry) error {
	bytes, err := json.Marshal(ent)
	if err != nil {
		log.Printf("EncodingError: LogEntry.Marshal fail: %+v", ent)
		return errmsg.EncodingError
	}
	if n, err := pw.saveRecord(bytes, LogEntry); err != nil {
		return err
	} else if n != len(bytes) {
		log.Printf("ShortWrite: SaveLogEntry, written=%d, len=%d", n, len(bytes))
		return errmsg.ShortWrite
	}
	return nil
}

func (pw *pageWriter) SaveState(state *core.PersistentState) error {
	bytes, err := json.Marshal(state)
	if err != nil {
		log.Printf("EncodingError: PersistentState.Marshal fail: %+v", state)
		return errmsg.EncodingError
	}
	if n, err := pw.saveRecord(bytes, State); err != nil {
		return err
	} else if n != len(bytes) {
		log.Printf("ShortWrite: SaveState, written=%d, len=%d", n, len(bytes))
		return errmsg.ShortWrite
	}
	//for _, ent := range state.Logs.GetLogList(0) {
	//	if err = pw.SaveLogEntry(ent); err != nil {
	//		return err
	//	}
	//}
	return nil
}

func (pw *pageWriter) sync() error {
	curOffset, err := pw.writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if curOffset < SegmentSizeBytes {
		return fs.Fsync(pw.writer)
	}
	return nil
}

func (pw *pageWriter) saveRecord(bytes []byte, dtype dataType) (int, error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	// fit the bytes into frames with fixed size
	writtenBytes := 0
	writtenFrameBytes := 0
	for writtenBytes < len(bytes) {
		leftPageSize := pw.pageSize - pw.offset%pw.pageSize
		leftDataSize := leftPageSize - FrameHeadSize
		if leftDataSize <= 0 {
			// not enough for a record
			padBytes := make([]byte, leftPageSize)
			if c, err := pw.writer.Write(padBytes); err != nil {
				return writtenBytes, err
			} else if c != leftPageSize {
				log.Printf("ShortWrite: saveRecordPad, written=%d, len=%d", c, leftPageSize)
				return writtenBytes, errmsg.ShortWrite
			} else {
				pw.offset += leftPageSize
			}
			continue
		}
		ftype := getFrameType(leftPageSize, len(bytes), writtenBytes)
		var bytesToWrite []byte
		if len(bytes)-writtenBytes <= leftDataSize {
			bytesToWrite = bytes[writtenBytes:]
		} else {
			bytesToWrite = bytes[writtenBytes : writtenBytes+leftDataSize]
		}
		var rec *Record
		var err error
		var n int
		if rec, err = NewRecord(bytesToWrite, dtype, ftype, pw.crc); err != nil {
			return writtenBytes, err
		}
		if n, err = pw.writeRecordToFile(rec); err != nil {
			return n, err
		}
		pw.offset += n
		writtenFrameBytes += n
		writtenBytes += n - FrameHeadSize
	}
	return writtenBytes, nil
}

func (pw *pageWriter) mustWriteRecordToFile(rec *Record) int {
	n, err := pw.writeRecordToFile(rec)
	if err != nil {
		panic(err)
	}
	return n
}

func (pw *pageWriter) writeRecordToFile(rec *Record) (int, error) {
	recBytes, err := rec.Marshal()
	if err != nil {
		return 0, errmsg.EncodingRecordError
	}
	if c, err := pw.writer.Write(recBytes); err != nil {
		return 0, err
	} else if c != len(recBytes) {
		log.Printf("ShortWrite: writeRecordToFile, written=%d, len=%d", c, len(recBytes))
		return c, errmsg.ShortWrite
	} else {
		return c, nil
	}
}

// getFrameType calculate the frame type and return
// availBytes is the bytes left within a page
// totalBytes is the total bytes of the record to be put into frames
// writtenBytes is the number of bytes already put into previous frames
func getFrameType(availBytes, totalBytes, doneBytes int) frameType {
	if doneBytes > 0 {
		if totalBytes-doneBytes+FrameHeadSize <= availBytes {
			return LastType
		} else {
			return MiddleType
		}
	} else {
		// new, no previous frame
		if totalBytes+FrameHeadSize <= availBytes {
			return FullType
		} else {
			return FirstType
		}
	}
}
