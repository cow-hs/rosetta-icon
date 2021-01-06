package mpt

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie"
)

func makePrefix(l, prefix int) []byte {
	if l <= 55 {
		return []byte{byte(prefix + l)}
	}

	prefix += 55
	bLen := 0
	tmp := l
	for {
		if tmp == 0 {
			break
		}
		tmp = tmp / 0x100
		bLen++
	}

	r := make([]byte, bLen+1)

	for i := range r {
		if i == 0 {
			r[0] = byte(prefix + bLen)
		} else {
			r[i] = byte(l >> uint(8*bLen) & 0xff)
		}
		bLen--
	}
	return r
}

func encodeByte(d []byte) []byte {
	l := len(d)
	if l == 0 {
		return []byte{0x80}
	}
	if l == 1 && d[0] < 0x80 {
		return d
	}
	return append(makePrefix(l, 0x80), d...)
}

func encodeList(data ...[]byte) []byte {
	r := make([]byte, 0)
	for _, d := range data {
		r = append(r, d...)
	}
	return append(makePrefix(len(r), 0xc0), r...)
}

var (
	errRLPNotEnoughBytes  = errors.New("RLP:Not enough bytes to decode")
	errRLPInvalidEncoding = errors.New("RLP:Invalid encoding")
)

func readSize(b []byte, slen byte) (uint64, error) {
	if int(slen) > len(b) {
		return 0, errRLPInvalidEncoding
	}
	var s uint64
	switch slen {
	case 1:
		s = uint64(b[0])
	case 2:
		s = uint64(b[0])<<8 | uint64(b[1])
	case 3:
		s = uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])
	case 4:
		s = uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
	case 5:
		s = uint64(b[0])<<32 | uint64(b[1])<<24 | uint64(b[2])<<16 | uint64(b[3])<<8 | uint64(b[4])
	case 6:
		s = uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 | uint64(b[3])<<16 | uint64(b[4])<<8 | uint64(b[5])
	case 7:
		s = uint64(b[0])<<48 | uint64(b[1])<<40 | uint64(b[2])<<32 | uint64(b[3])<<24 | uint64(b[4])<<16 | uint64(b[5])<<8 | uint64(b[6])
	case 8:
		s = uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 | uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
	}
	if s < 56 || b[0] == 0 {
		return 0, errRLPInvalidEncoding
	}
	return s, nil
}

func getContentSize(buf []byte) (uint64, uint64, error) {
	if len(buf) == 0 {
		return 0, 0, errRLPNotEnoughBytes
	}
	b := buf[0]
	var tagsize uint64
	var contentsize uint64
	var err error
	switch {
	case b < 0x80:
		tagsize = 0
		contentsize = 1
	case b < 0xB8:
		tagsize = 1
		contentsize = uint64(b - 0x80)
		if contentsize == 1 && len(buf) > 1 && buf[1] < 128 {
			return 0, 0, errRLPInvalidEncoding
		}
	case b < 0xC0:
		tagsize = uint64(b-0xB7) + 1
		contentsize, err = readSize(buf[1:], b-0xB7)
	case b < 0xF8:
		tagsize = 1
		contentsize = uint64(b - 0xC0)
	default:
		tagsize = uint64(b-0xF7) + 1
		contentsize, err = readSize(buf[1:], b-0xF7)
	}
	if err != nil {
		return 0, 0, err
	}

	if contentsize > uint64(len(buf))-tagsize {
		return 0, 0, errRLPNotEnoughBytes
	}
	return tagsize, contentsize, err
}

func decodeValue(buf []byte, t reflect.Type, db db.Database) trie.Object {
	if t == reflect.TypeOf([]byte{}) {
		return byteValue(buf)
	}
	vobj := reflect.New(t.Elem())
	nobj, ok := vobj.Interface().(trie.Object)
	if !ok {
		panic("Failed to decode")
		return nil
	}
	if err := nobj.Reset(db, buf); err != nil {
		panic("Failed to decode")
		return nil
	}
	return nobj
}

func decodeBranch(buf []byte, t reflect.Type, db db.Database) node {
	// serialized branch can have list which is another branch(sharedNibbles/value) or a leaf(keyEnd/value) or  hexa(serialized(rlp))
	tagSize, contentSize, _ := getContentSize(buf)
	// child is leaf, hash or nil(128)
	newBranch := &branch{nodeBase: nodeBase{state: committedNode, serializedValue: buf}}
	for i, valueIndex := tagSize, 0; i < tagSize+contentSize; valueIndex++ {
		tagSize, contentSize, _ := getContentSize(buf[i:])
		buf := buf[i:]
		if valueIndex == 16 {
			// value of branch is not hashed
			if contentSize == 0 {
				newBranch.value = nil
			} else {
				newBranch.value = decodeValue(buf[tagSize:tagSize+contentSize], t, db)
			}
		} else {
			// hash node
			if contentSize == 0 {
				newBranch.nibbles[valueIndex] = nil
			} else {
				if hashableSize == contentSize {
					newBranch.nibbles[valueIndex] = hash(buf[tagSize : tagSize+contentSize])
				} else {
					newBranch.nibbles[valueIndex] = deserialize(buf[tagSize:tagSize+contentSize], t, db)
				}
			}
		}

		i += tagSize + contentSize
	}
	return newBranch
}

// even : 00 or 20 bit sequence
// odd : 1X or 3X bit sequence

//0        0000    |       extension              even
//1        0001    |       extension              odd
//2        0010    |   terminating (leaf)         even
//3        0011    |   terminating (leaf)         odd

// get first nibble and check if 0x2 | nibble is true, leaf. if not, extension
//2nd bit is 1, leaf
// if nodeType is 0, extension. leaf is 1
func decodeKey(buf []byte) (keyBuf []byte, nodeType int, err error) {
	firstNib := buf[0] >> 4
	index := 0

	nodeType = 0
	if firstNib&0x2 == 0x2 {
		nodeType = 1
	}
	if firstNib%2 == 0 { // even. first byte is just padding byte
		keyBuf = make([]byte, (len(buf)-1)*2)
	} else { // odd
		keyBuf = make([]byte, (len(buf)*2 - 1))
		keyBuf[0] = buf[0] & 0x0F
		index = 1
	}

	buf = buf[1:]
	for i := 0; i < len(buf); i++ {
		keyBuf[i*2+index] = buf[i] >> 4
		keyBuf[i*2+1+index] = buf[i] & 0x0F
	}
	return keyBuf, nodeType, nil
}

func deserialize(b []byte, t reflect.Type, db db.Database) node {
	listTagsize, _, _ := getContentSize(b) // length of list tag
	list := b[listTagsize:]
	var keyBuf []byte
	var valBuf []byte

	for i := 0; len(list) > 0; i++ {
		// 1. list(0xC0) exists
		// 2. loop count is bigger than 2
		// then it's branch
		if 2 <= i {
			return decodeBranch(b, t, db)
		}

		tagsize, size, err := getContentSize(list) // length of byte
		if err != nil {
			return nil
		}
		if len(keyBuf) == 0 {
			keyBuf = list[tagsize : tagsize+size]
		} else {
			if 0xC0 <= list[0] {
				valBuf = list[:tagsize+size]
			} else {
				valBuf = list[tagsize : tagsize+size]
			}
		}
		list = list[tagsize+size:]
	}

	nodeType := 0
	keyBuf, nodeType, _ = decodeKey(keyBuf)
	if nodeType == 0 { //extension
		if hashableSize > len(valBuf) {
			return &extension{sharedNibbles: keyBuf, next: decodeBranch(valBuf, t, db),
				nodeBase: nodeBase{serializedValue: b, state: committedNode}}
		}
		return &extension{sharedNibbles: keyBuf, next: hash(valBuf),
			nodeBase: nodeBase{serializedValue: b, state: committedNode}}
	}
	return &leaf{keyEnd: keyBuf, value: decodeValue(valBuf, t, db),
		nodeBase: nodeBase{serializedValue: b, state: committedNode}}
}
