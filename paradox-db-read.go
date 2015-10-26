package main

import (
	"os"
	"time"
	"bytes"
	"strings"
	"io/ioutil"
	"encoding/binary"
	"github.com/paulrosania/go-charset/charset"
	_ "github.com/paulrosania/go-charset/data"
)

const (
	offsetNumFields     = 0x21
	offsetFieldTypeSize = 0x78

	PX_Field_Type_Alpha       = 0x01
	PX_Field_Type_Date        = 0x02
	PX_Field_Type_ShortInt    = 0x03
	PX_Field_Type_LongInt     = 0x04
	PX_Field_Type_Currency    = 0x05
	PX_Field_Type_Number      = 0x06
	PX_Field_Type_Logical     = 0x09
	PX_Field_Type_MemoBLOB    = 0x0c
	PX_Field_Type_BinBLOB     = 0x0d
	PX_Field_Type_Graphic     = 0x10
	PX_Field_Type_Time        = 0x14
	PX_Field_Type_Timestamp   = 0x15
	PX_Field_Type_Incremental = 0x16
	PX_Field_Type_BCD         = 0x17
)

type (
	header struct {
		recordSize uint16
		headerSize uint16
		numFields  uint16
	}

	field struct {
		name string
		typ  uint8
		size uint8
	}

	block struct {
		nextBlock uint16
		prevBlock uint16
		addSize   uint16
		numRecs   uint16
		records   uint16
	}

	paradoxTable interface {
		appendRow(values ...interface{})
	}
)

func paradoxReadTable(path string, table paradoxTable, fixHour int) error {
	// Read file in slice
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var h header
	r := bytes.NewReader(data)

	// RecordSize
	binary.Read(r, binary.LittleEndian, &h.recordSize)
	// HeaderSize
	binary.Read(r, binary.LittleEndian, &h.headerSize)
	// Num Fields in table
	r.Seek(offsetNumFields, os.SEEK_SET)
	binary.Read(r, binary.LittleEndian, &h.numFields)

	// Get data for fields
	var fields []field
	r.Seek(offsetFieldTypeSize, os.SEEK_SET)
	for i := 0; i < int(h.numFields); i++ {
		var f field
		binary.Read(r, binary.LittleEndian, &f.typ)
		binary.Read(r, binary.LittleEndian, &f.size)
		fields = append(fields, f)
	}

	// Skip trash data: tablenameptr, fieldnameptr
	r.Seek(int64(4+4*h.numFields+261), os.SEEK_CUR)

	// Get fields name
	for i := 0; i < int(h.numFields); i++ {
		var name string
		for {
			b, _ := r.ReadByte()
			if b == 0x00 { // delimiter
				break
			}
			name += string(b)
		}
		fields[i].name = string(name)
	}

	// For blocks of data in file
	nblock := int64(1)
	for {
		r.Seek(int64(h.headerSize)*nblock, os.SEEK_SET)
		nblock++
		var bl block
		err = binary.Read(r, binary.LittleEndian, &bl.nextBlock)
		if err != nil { // EOF
			break
		}
		binary.Read(r, binary.LittleEndian, &bl.prevBlock)
		binary.Read(r, binary.LittleEndian, &bl.addSize)

		if bl.addSize > 32767 {
			bl.addSize = 0
			bl.numRecs = 0
		} else {
			bl.numRecs = (bl.addSize / h.recordSize) + 1
		}

		// For records in blocks
		for i := 0; i < int(bl.numRecs); i++ {
			// For fields in record
			curRecSize := 0
			var vals []interface{}
			for j := 0; j < int(h.numFields); j++ {
				typ := fields[j].typ
				size := fields[j].size
				curRecSize += int(size)
				sbuf := make([]byte, 1024)
				r.Read(sbuf[:size])
				switch typ {
				case PX_Field_Type_LongInt, PX_Field_Type_Incremental, PX_Field_Type_Number:
					if sbuf[0] == 0x80 {
						sbuf[0] = 0x00
					}
					vals = append(vals, int(intBE32(sbuf[:size])))
				case PX_Field_Type_Date:
					vals = append(vals, "")
				case PX_Field_Type_Timestamp:
					sbuf[0] &= 0x7f
					var timestamp int64
					timestamp = int64(intBE64(sbuf[:size]))
					timestamp >>= 8
					timestamp /= 500
					timestamp -= 37603860709183
					timestamp -= 10800 + int64(fixHour*60*60) // fix hours
					vals = append(vals, time.Unix(timestamp, 0))
				default:
					sbuf = bytes.Trim(sbuf, "\x00")
					size = uint8(len(sbuf))
					r, _ := charset.NewReader("windows-1251", strings.NewReader(string(sbuf[:size])))
					result, _ := ioutil.ReadAll(r)
					vals = append(vals, string(result))
				}
			}
			table.appendRow(vals...)
			if curRecSize < int(h.recordSize) {
				r.Seek(int64(int(h.recordSize)-curRecSize), os.SEEK_CUR)
			}
		}
	}
	return nil
}

func intBE32(i []byte) uint32 {
	return binary.BigEndian.Uint32(i)
}

func intBE64(i []byte) uint64 {
	return binary.BigEndian.Uint64(i)
}
