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
	"io/ioutil"
	"os"
	"testing"
	"time"
	"unsafe"
)

const (
	TestJpegFile    = "test_files/big_endian.jpg"
	TestJpegOutFile = "test_files/test_jpeg_compressed.jpg"
)

func isHostLittleEndian() bool {
	// From https://groups.google.com/forum/#!topic/golang-nuts/3GEzwKfRRQw
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	return (b == 0x04)
}

func TestRawParsers(t *testing.T) {
	rp := NewRawParsers()

	if rp == nil {
		t.Fail()
	}

	// nef parser
	nefparser, key := NewNefParser(isHostLittleEndian())
	if nefparser == nil || key != NefParserKey {
		t.Fail()
	}
	rp.Register(NefParserKey, nefparser)
	if rp.GetParser(NefParserKey) == nil {
		t.Fail()
	}
	// delete parser
	rp.DeleteParser(NefParserKey)

	// ensure deleted
	if rp.GetParser(NefParserKey) != nil {
		t.Fail()
	}

	// cr2 parser
	cr2parser, key := NewCr2Parser(isHostLittleEndian())
	if cr2parser == nil || key != Cr2ParserKey {
		t.Fail()
	}
	rp.Register(Cr2ParserKey, cr2parser)
	if rp.GetParser(Cr2ParserKey) == nil {
		t.Fail()
	}
	// delete parser
	rp.DeleteParser(Cr2ParserKey)

	// ensure deleted
	if rp.GetParser(Cr2ParserKey) != nil {
		t.Fail()
	}

	// test non-existent parser
	if rp.GetParser("") != nil {
		t.Fail()
	}

}

func TestBytesToUShort(t *testing.T) {
	if isHostLittleEndian() {
		var leInt, leResult, beInt, beResult uint16
		dataLe := []byte{0xBB, 0xAA}

		// values based on little/big endian
		leInt = 0xAABB
		beInt = 0xBBAA

		leResult = bytesToUShort(isHostLittleEndian(), false, dataLe)
		t.Logf("Little Endian Result: 0x%02x", leResult)
		if leInt != leResult {
			t.Fatalf("Conversion failed.  Expected 0x%x Got: 0x%x\n",
				leInt, leResult)

		}

		beResult = bytesToUShort(isHostLittleEndian(), true, dataLe)
		t.Logf("Big Endian Result: 0x%02x", beResult)
		if beInt != beResult {
			t.Fatalf("Conversion failed.  Expected 0x%x Got: 0x%x\n",
				beInt, beResult)
		}
	}
}

func TestBytesToUInt(t *testing.T) {
	if isHostLittleEndian() {
		var leInt, leResult, beInt, beResult uint32
		dataLe := []byte{0xDD, 0xCC, 0xBB, 0xAA}

		// values based on little/big endian
		leInt = 0xAABBCCDD
		beInt = 0xDDCCBBAA

		leResult = bytesToUInt(isHostLittleEndian(), false, dataLe)
		t.Logf("Little Endian Result: 0x%02x", leResult)
		if leInt != leResult {
			t.Fatalf("Conversion failed.  Expected 0x%x Got: 0x%x\n",
				leInt, leResult)

		}

		beResult = bytesToUInt(isHostLittleEndian(), true, dataLe)
		t.Logf("Big Endian Result: 0x%02x", beResult)
		if beInt != beResult {
			t.Fatalf("Conversion failed.  Expected 0x%x Got: 0x%x\n",
				beInt, beResult)
		}
	}
}

func TestBytesToAsciiString(t *testing.T) {
	bytes := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x21}
	result := bytesToASCIIString(bytes)
	if result != "Hello!" {
		t.Fail()
	}
}

func TestNumericMonthToRfc822(t *testing.T) {
	validMonths := []string{
		"01",
		"02",
		"03",
		"04",
		"05",
		"06",
		"07",
		"08",
		"09",
		"10",
		"11",
		"12",
	}

	invalidMonths := []string{
		"",
		"00",
		"13",
		"-1",
		"aa",
	}

	for i, d := range validMonths {
		t.Logf("Month numeric %s i=%d", d, i)
		tokens := []string{"2010", d, "10"}
		result, e := toRfc822Date(tokens)
		if e != nil {
			t.Fail()
		}
		fail := true
		switch i {
		case 0:
			if result == "Jan" {
				fail = false
			}
		case 1:
			if result == "Feb" {
				fail = false
			}
		case 2:
			if result == "Mar" {
				fail = false
			}
		case 3:
			if result == "Apr" {
				fail = false
			}
		case 4:
			if result == "May" {
				fail = false
			}
		case 5:
			if result == "Jun" {
				fail = false
			}
		case 6:
			if result == "Jul" {
				fail = false
			}
		case 7:
			if result == "Aug" {
				fail = false
			}
		case 8:
			if result == "Sep" {
				fail = false
			}
		case 9:
			if result == "Oct" {
				fail = false
			}
		case 10:
			if result == "Nov" {
				fail = false
			}
		case 11:
			if result == "Dec" {
				fail = false
			}
		}
		if fail {
			t.Fail()
		}
	}

	for i, d := range invalidMonths {
		t.Logf("Month numeric %s i=%d", d, i)
		tokens := []string{"2010", d, "10"}
		_, e := toRfc822Date(tokens)
		t.Logf("Expected error: %v\n", e)
		if e == nil {
			t.Fail()
		}
	}
}

func TestParseDateTime(t *testing.T) {
	dateTime := "2010:08:10 12:11:07"
	parsedTime, e := parseDateTime(dateTime)
	if e != nil {
		t.Fatalf("Unexpected error parsing date and time: %v\n", e)
	} else {
		const format = "02 Jan 06 15:04"
		refTime, e := time.Parse(format, "10 Aug 10 12:11")
		if e != nil || !refTime.Equal(parsedTime) {
			t.Fail()
		}
	}
}

func TestParseTimeInvalid(t *testing.T) {
	// invalid month
	dateTime := "2010:13:10 12:11:07"
	_, err := parseDateTime(dateTime)
	if err == nil {
		t.Fail()
	}

	// empty string
	dateTime = ""
	_, err = parseDateTime(dateTime)
	if err == nil {
		t.Fail()
	}

	// invalid time
	dateTime = "2010:12:10 AA:BB:CC"
	_, err = parseDateTime(dateTime)
	if err == nil {
		t.Fail()
	}

	// invalid time (2 tokens only)
	dateTime = "2010:12:10 12:11"
	_, err = parseDateTime(dateTime)
	if err == nil {
		t.Fail()
	}

	// invalid date (2 tokens only)
	dateTime = "2010:12 12:11:07"
	_, err = parseDateTime(dateTime)
	if err == nil {
		t.Fail()
	}
}

func TestJpegCodec(t *testing.T) {
	var err error
	data, err := ioutil.ReadFile(TestJpegFile)
	if err != nil {
		t.Fatalf("Error reading file: %v\n", err)
	}
	t.Logf("Read %d bytes from file\n", len(data))

	for i := 0; i < 1; i++ {
		err = decodeAndWriteJpeg(data, 75, TestJpegOutFile)
		defer os.Remove(TestJpegOutFile)
		if err != nil {
			t.Errorf("Error while decode and write jpeg: %v\n", err)
		}

		// verify jpeg has been compressed
		info, err := os.Stat(TestJpegOutFile)
		if err != nil {
			t.Errorf("Compressed jpeg not created: %v\n", err)
		}
		t.Logf("Compressed jpeg details: %v\n", info)
		if info.Size() == 0 {
			t.Fail()
		}
	}
}
