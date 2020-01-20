package main

import (
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	d, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	x, err := parseDescriptor(d)
	if err != nil {
		log.Fatal(err)
	}
	v, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(v)
}

// lowest level: wire encoding

// https://developers.google.com/protocol-buffers/docs/proto#simple
type tagNum = uint32 // just 30 bit really - and 0 is invalid

type tagClass = byte

// https://developers.google.com/protocol-buffers/docs/encoding
const (
	tagUvarint  tagClass = 0 // int32, int64, uint32, uint64, sint32, sint64, bool, enum
	tag64bit    tagClass = 1 // fixed64, sfixed64, double
	tagSequence tagClass = 2 // length prefixed bytes: string, bytes, embedded message, packed repeated fields
	tagStart    tagClass = 3 // start group - deprecated
	tagEnd      tagClass = 4 // end group - deprecated
	tag32bit    tagClass = 5 // fixed32, sfixed32, float
)

// readNext reads the next tag.
// Errors are encoded by next <= 0 and kind will be contained in d.
// next == 0 if data is too short.
// next == -(bytes read) if data is invalid.
func readNext(data []byte) (d uint64, b []byte, tag tagNum, next int) {
	// TODO use unsafe assembler optimistically and aggressively to avoid slow-paths?
	// read after reserved memory, avoid bounds-checking, ...?
	v, pos := binary.Uvarint(data)
	tag = tagNum(v >> 3) // valid iff tag > 0 && tag < ((1<<30) - 1)
	kind := tagClass(v & 0x07)
	if tag == 0 {
		return uint64(kind), nil, tag, pos
	}
	next = pos
	switch kind {
	case tagUvarint:
		v, pos := binary.Uvarint(data[next:])
		if pos < 0 {
			break
		}
		return v, nil, tag, next + int(pos)
	case tag32bit:
		start := next
		next += 4
		v := binary.LittleEndian.Uint32(data[start:next])
		return uint64(v), nil, tag, next
	case tag64bit:
		start := next
		next = next + 8
		v := binary.LittleEndian.Uint64(data[start:next])
		return v, nil, tag, next
	case tagSequence:
		v, pos := binary.Uvarint(data[next:])
		if pos == 0 {
			break
		}
		start := next + pos
		next = start + int(v)
		return 0, data[start:next:next], tag, next
	default:
	}
	// error, report kind and tag
	return uint64(kind), nil, tag, 0
}

type File struct {
	Name    string     `json:",omitempty"` // 1
	Package string     `json:",omitempty"` // 2
	Message []*Message `json:",omitempty"` // 4
	Format  string     `json:",omitempty"` // 12
}

type Message struct {
	Name   string     `json:",omitempty"` // 1
	Field  []*Field   `json:",omitempty"` // 2
	Nested []*Message `json:",omitempty"` // 4
}

type Field struct {
	Name       string `json:",omitempty"` // 1
	Tag        tagNum `json:",omitempty"` // 3
	Label      uint8  `json:",omitempty"` // 4
	Type       uint8  `json:",omitempty"` // 5
	OneOfIndex int32  `json:",omitempty"` // 9
}

type badOffset int

func (err *badOffset) Error() string {
	return "incomplete proto"
}

func parseDescriptor(msg []byte) ([]*File, error) {
	var files []*File
	for i := 0; i < len(msg); {
		_, b, t, n := readNext(msg[i:])
		if n == 0 {
			tmp := badOffset(i)
			return files, &tmp
		}
		switch t {
		case 1:
			f, err := parseFile(b)
			if err != nil {
				tmp := badOffset(i) + *err
				return files, &tmp
			}
			files = append(files, f)
		default: // skip
		}
		i += n
	}
	return files, nil
}

func parseFile(msg []byte) (*File, *badOffset) {
	f := &File{}
	for i := 0; i < len(msg); {
		_, b, t, n := readNext(msg[i:])
		if n == 0 {
			tmp := badOffset(i)
			return f, &tmp
		}
		switch t {
		case 1:
			f.Name = string(b)
		case 2:
			f.Package = string(b)
		case 4:
			m, err := parseMessage(b)
			if err != nil {
				tmp := badOffset(i) + *err
				return f, &tmp
			}
			f.Message = append(f.Message, m)
		case 12:
			f.Format = string(b)
		default: // skip
		}
		i += n
	}
	return f, nil
}

func parseMessage(msg []byte) (*Message, *badOffset) {
	m := &Message{}
	for i := 0; i < len(msg); {
		_, b, t, n := readNext(msg[i:])
		if n == 0 {
			tmp := badOffset(i)
			return m, &tmp
		}
		switch t {
		case 1:
			m.Name = string(b)
		case 2:
			f, err := parseField(b)
			if err != nil {
				tmp := badOffset(i) + *err
				return m, &tmp
			}
			m.Field = append(m.Field, f)
		case 4:
			nm, err := parseMessage(b)
			if err != nil {
				tmp := badOffset(i) + *err
				return m, &tmp
			}
			m.Nested = append(m.Nested, nm)
		default: // skip
		}
		i += n
	}
	return m, nil
}

func parseField(msg []byte) (*Field, *badOffset) {
	f := &Field{}
	for i := 0; i < len(msg); {
		d, b, t, n := readNext(msg[i:])
		if n == 0 {
			tmp := badOffset(i)
			return f, &tmp
		}
		switch t {
		case 1:
			f.Name = string(b)
		case 3:
			f.Tag = uint32(d)
		case 4:
			f.Label = uint8(d) // labelType
		case 5:
			f.Type = uint8(d) // tagClass
		case 9:
			f.OneOfIndex = int32(d)
		default: // skip
		}
		i += n
	}
	return f, nil
}
