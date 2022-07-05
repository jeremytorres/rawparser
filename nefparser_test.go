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
	"os"
	"testing"
)

const (
	TestNefFile       = "test_files/big_endian.NEF"
	TestNefNoJpegFile = "test_files/little_endian_no_jpeg.NEF"
)

var (
	gHostIsLe  bool
	gNefParser *NefParser
)

func setupNef() {
	gHostIsLe = isHostLittleEndian()
	gNefParser = &NefParser{&rawParser{gHostIsLe}}
}

func openTestNefFile() (*os.File, error) {
	return os.Open(TestNefFile)
}

func getNefHeader(f *os.File) (*nefHeader, error) {
	header, err := gNefParser.processHeader(f)
	return header, err
}

func getNefTestDir() (string, error) {
	curdir, e := os.Getwd()
	if e != nil {
		return "", e
	}
	testdir := curdir + string(os.PathSeparator) + "test_files" + string(os.PathSeparator)
	return testdir, nil
}

func TestNewNefParserInstance(t *testing.T) {
	setupNef()

	// flag indicating host is big endian
	instance1, _ := NewNefParser(false)

	// flag indicating host is little endian
	instance2, _ := NewNefParser(true)

	if instance1 == nil || instance2 == nil {
		t.Fail()
	}
}

func TestProcessNefHeader(t *testing.T) {
	setupNef()

	f, e := openTestNefFile()
	if e == nil {
		defer f.Close()
		h, err := getNefHeader(f)
		if err != nil {
			t.Logf("Error: %v\n", err)
			t.Fail()
		}
		t.Logf("Header: %v\n", h)
		if h.isBigEndian && h.tiffMagicValue == 42 &&
			h.tiffOffset == 8 {
			t.Log("Valid Nef header big endian.")

		} else {
			t.Fail()
		}
	} else {
		t.Fatalf("Unable to open test NEF file: %v\n", e)
	}
}

func TestProcessNefIfds(t *testing.T) {
	setupNef()

	// big endian nef
	f, e := openTestNefFile()
	if e == nil {
		defer f.Close()
		h, err := getNefHeader(f)
		if err != nil {
			t.Fail()
		}
		t.Logf("Header: %v\n", h)
		jpegInfo, createDate, err := gNefParser.processIfds(f, h)
		if err != nil {
			t.Fail()
		}
		t.Logf("jegInfo: %v createDate: %v\n", jpegInfo, createDate)

	} else {
		t.Fatalf("Unable to open test NEF file: %v\n", e)
	}
}

func TestProcessNefJpegDecodeAndWrite(t *testing.T) {
	setupNef()

	f, e := openTestNefFile()
	if e == nil {
		defer f.Close()
		h, err := getNefHeader(f)
		if err != nil {
			t.Fail()
		}
		t.Logf("Header: %v\n", h)
		jpegInfo, createDate, err := gNefParser.processIfds(f, h)
		if err != nil {
			t.Fail()
		}
		t.Logf("jpegInfo: %v createDate: %v\n", jpegInfo, createDate)

		curdir, e := os.Getwd()
		if e != nil {
			t.Fatal("Unabled get get current directory")
		}
		testdir := curdir + string(os.PathSeparator) + "test_files" + string(os.PathSeparator)
		t.Logf("Test dir: %s\n", testdir)
		jpegPath, err := gNefParser.decodeAndWriteJpeg(f, jpegInfo, testdir, 50)
		if err != nil {
			t.Fail()
		}
		defer os.Remove(jpegPath)
		t.Logf("Extracted jpeg path: %v\n", jpegPath)

		// verify jpeg has been extracted
		info, e := os.Stat(jpegPath)
		if e != nil {
			t.Fail()
		}
		t.Logf("Extracted jpeg details: %v\n", info)
		if info.Size() == 0 {
			t.Fail()
		}
	} else {
		t.Fatalf("Unable to open test NEF file: %v\n", e)
	}
}

func TestNefProcessFile(t *testing.T) {
	setupNef()

	testdir, e := getNefTestDir()
	if e == nil {
		// big endian nef
		ni := RawFileInfo{TestNefFile, testdir, 50}
		nef, err := gNefParser.ProcessFile(&ni)
		defer os.Remove(nef.JpegPath)
		if err != nil {
			t.Fatal("Unexpected error while parsing test big endian NEF")
		}
		// verify jpeg has been extracted
		info, e := os.Stat(nef.JpegPath)
		if e != nil {
			t.Fail()
		}
		t.Logf("Extracted jpeg details: %v\n", info)
		if info.Size() == 0 {
			t.Fail()
		}
		t.Logf("Parsed big endian Nef: %v\n", nef)
	}
}

func TestNefProcessFileNoJpeg(t *testing.T) {
	setupNef()

	testdir, e := getNefTestDir()
	if e == nil {
		ni := RawFileInfo{TestNefNoJpegFile, testdir, 50}
		_, err := gNefParser.ProcessFile(&ni)
		if err == nil {
			t.Fail()
		}
	} else {
		t.Fatal("Unable to determine test directory")
	}
}

func TestNefProcessNonExistentFile(t *testing.T) {
	setupNef()

	testdir, e := getNefTestDir()
	if e != nil {
		t.Fatal("Unable to determine test directory")
	} else {
		ni := RawFileInfo{"", testdir, 50}
		_, err := gNefParser.ProcessFile(&ni)
		if err == nil {
			t.Fatal("Expected error not generated while parsing NEF")
		} else {
			t.Logf("Received expected error: %v\n", err)
		}
	}
}

func TestNefEndianessState(t *testing.T) {
	setupNef()

	if gNefParser.SetHostIsLittleEndian(true); gNefParser.IsHostLittleEndian() != true {
		t.Fail()
	}

	if gNefParser.SetHostIsLittleEndian(false); gNefParser.IsHostLittleEndian() != false {
		t.Fail()
	}
}
