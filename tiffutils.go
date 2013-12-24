/*
 Copyright (c) 2013 Jeremy Torres, https://github.com/jeremytorres/rawparser

 Permission is hereby granted, free of charge, to any person obtaining
 a copy of this software and associated documentation files (the
 "Software"), to deal in the Software without restriction, including
 without limitation the rights to use, copy, modify, merge, publish,
 distribute, sublicense, and/or sell copies of the Software, and to
 permit persons to whom the Software is furnished to do so, subject to
 the following conditions:

 The above copyright notice and this permission notice shall be
 included in all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
 LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
 OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
 WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package rawparser

import (
	"container/list"
	"fmt"
	"os"
)

// bytesToUShort is a utility function for converting bytes
// representing an unsigned short, based on a raw file's defined
// endianess.
// isBigEndian is an input parameter defining the raw file endiannes.
// Returns an uint16 based on the raw file endianness.
//
// Implemenation Note: to reduce the error handling code,
// the critical function for retrieving bytes is error checked. Therefore,
// it's assumed the caller will supply exactly 2 bytes.
func bytesToUShort(isHostLittleEndian, isBigEndian bool, buf []byte) uint16 {
	var val uint16

	if isBigEndian && isHostLittleEndian {
		val = (uint16(buf[0]) << 8) | (uint16(buf[1]) & 0xFF)
	} else {
		val = (uint16(buf[1]) << 8) | (uint16(buf[0]) & 0xFF)
	}

	return val
}

// bytesToUInt is a utility function for converting bytes
// representing an unsigned int, based on a raw file's defined
// endianess.
// isBigEndian is an input parameter defining the raw file endiannes.
// Returns an uint32 based on the raw file endianness.
//
// Implemenation Note: to reduce the error handling code,
// the critical function for retrieving bytes is error checked. Therefore,
// it's assumed the caller will supply exactly 4 bytes.
func bytesToUInt(isHostLittleEndian, isBigEndian bool, buf []byte) uint32 {
	var a, b uint16
	var val uint32

	a = bytesToUShort(isHostLittleEndian, isBigEndian, buf[0:2])
	b = bytesToUShort(isHostLittleEndian, isBigEndian, buf[2:])

	if isBigEndian && isHostLittleEndian {
		// convert
		val = (uint32(a)<<16 | (uint32(b) & 0xFFFF))
	} else {
		val = (uint32(b)<<16 | (uint32(a) & 0xFFFF))
	}

	return val
}

// bytesToAsciiString is a utility function for converting bytes
// to an ASCII string.  Returns a new string given the ASCII bytes.
func bytesToASCIIString(bytes []byte) (val string) {
	val = string(bytes[:])
	return val
}

// readField reads a specified number of bytes from the raw file based
// on an offset.  Returns the bytes read or error.
func readField(offset int64, bytesToRead uint32, f *os.File) (bytes []byte, err error) {
	cache := make([]byte, bytesToRead)

	bytesRead, err := f.ReadAt(cache, int64(offset))
	if bytesRead != int(bytesToRead) {
		err = fmt.Errorf("read %d bytes; expected %d\n", bytesRead, bytesToRead)
	}

	return cache, err
}

// processIfd processed a TIFF IFD, based on:
// the parsed raw file header and a given offset witin the raw file.
// Returns a list of processed IFDs or error.
func processIfd(isHostLe, isFileBe bool, offset int64, f *os.File) (*list.List, error) {
	l := list.New()

	// entries
	bytes, err := readField(offset, 2, f)
	//	log.Printf("Bytes: %v\n", bytes)
	entries := bytesToUShort(isHostLe, isFileBe, bytes)
	//	log.Printf("Entries in IFD0: 0x%x\n", entries)
	offset += 2

	for i := 0; i < int(entries); i++ {
		var entry ifdEntry
		// tag
		bytes, err = readField(offset, 2, f)
		if err != nil {
			return l, err
		}
		entry.tag = bytesToUShort(isHostLe, isFileBe, bytes)
		offset += 2

		// type
		bytes, err = readField(offset, 2, f)
		if err != nil {
			return l, err
		}
		entry.fieldType = bytesToUShort(isHostLe, isFileBe, bytes)
		offset += 2

		// count
		bytes, err = readField(offset, 4, f)
		if err != nil {
			return l, err
		}
		entry.count = bytesToUInt(isHostLe, isFileBe, bytes)
		offset += 4

		// value offset
		bytes, err = readField(offset, 4, f)
		if err != nil {
			return l, err
		}
		entry.valueOffset = bytesToUInt(isHostLe, isFileBe, bytes)
		if err != nil {
			return l, err
		}
		offset += 4

		l.PushBack(entry)
	}

	return l, err
}

// processRationalEntry determines a TIFF-based rational entry (fractional) for
// per a given offset and raw file header.
// Returns a numerator, denominator, and rational (fractional) value or error.
func processRationalEntry(isHostLe, isFileBe bool, offset uint32, f *os.File) (num, den uint32, r float64, err error) {
	// numerator
	bytes, err := readField(int64(offset), 4, f)
	num = bytesToUInt(isHostLe, isFileBe, bytes)

	// denominator
	bytes, err = readField(int64(offset)+4, 4, f)
	den = bytesToUInt(isHostLe, isFileBe, bytes)

	if den > 0 {
		r = float64(num / den)
	} else {
		r = 0
	}

	return num, den, r, err
}

// processAsciiEntry converts a TIFF-based ASCII entry into a string
// per a given offset and raw file header.
// Return a string based on the ASCII bytes.
func processASCIIEntry(entry *ifdEntry, f *os.File) (val string, err error) {
	bytes, err := readField(int64(entry.valueOffset), entry.count, f)
	val = bytesToASCIIString(bytes)

	return val, err
}

// processShortValue extracts a 16-bit (unsigned short) value from a
// 4-bytes.  Per the TIFF spec, a tag with type 3 (unsigned short) will
// contain a left-justified value within a 4-bytes value offset.
// Returns an uint16.
func processShortValue(isFileBe bool, val uint32) (r uint16) {
	// assume big endian: msb/lsb
	msb, lsb := (val >> 16), (val & 0x0000FFFF)
	if isFileBe {
		r = uint16(msb)
	} else {
		r = uint16(lsb)
	}

	return r
}
