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
	TestCR2File = "test_files/little_endian.CR2"
)

var (
	gCr2Parser *Cr2Parser
)

func setupCr2() {
	gHostIsLe = isHostLittleEndian()
	gCr2Parser = &Cr2Parser{&rawParser{gHostIsLe}}
}

func openTestCr2File() (*os.File, error) {
	return os.Open(TestCR2File)
}

func getCr2Header(f *os.File) (*cr2Header, error) {
	header, err := gCr2Parser.processHeader(f)
	return header, err
}

func getCr2TestDir() (string, error) {
	curdir, e := os.Getwd()
	if e != nil {
		return "", e
	}
	testdir := curdir + string(os.PathSeparator) + "test_files" + string(os.PathSeparator)
	return testdir, nil
}

func TestNewCR2ParserInstance(t *testing.T) {
	setupCr2()

	// flag indicating host is big endian
	instance1, _ := NewCr2Parser(false)

	// flag indicating host is little endian
	instance2, _ := NewCr2Parser(true)

	if instance1 == nil || instance2 == nil {
		t.Fail()
	}
}

func TestCr2ProcessHeader(t *testing.T) {
	setupCr2()

	// little endian test CR2
	f, e := openTestCr2File()
	defer f.Close()
	if e == nil {
		h, err := getCr2Header(f)
		if err != nil {
			t.Fail()
		}
		t.Logf("Header: %v\n", h)
		if !h.isBigEndian && h.tiffMagicValue == 42 &&
			h.tiffOffset == 16 && h.cr2MagicValue == "CR" {
			t.Logf("Valid CR2 header parsed for little endian.")
		} else {
			t.Fail()
		}
	} else {
		t.Fatalf("Unable to open test CR2 file: %v\n", e)
	}
}

func TestCr2ProcessIfds(t *testing.T) {
	setupCr2()

	// little endian CR2
	f, e := openTestCr2File()
	defer f.Close()
	if e == nil {
		h, err := getCr2Header(f)
		if err != nil {
			t.Fail()
		}
		t.Logf("Header: %v\n", h)
		jpegInfo, createDate, err := gCr2Parser.processIfds(f, h)
		if err != nil {
			t.Errorf("Error processing IFDs: %v\n", err)
		}
		t.Logf("jpegInfo: %v createDate: %v\n", jpegInfo, createDate)

	} else {
		t.Fatalf("Unable to open test CR2 file: %v\n", e)
	}
}

func TestProcessJpegDecodeAndWrite(t *testing.T) {
	setupCr2()

	// little endian CR2
	f, e := openTestCr2File()
	defer f.Close()
	if e == nil {
		h, err := getCr2Header(f)
		if err != nil {
			t.Fail()
		}
		t.Logf("Header: %v\n", h)
		jpegInfo, createDate, err := gCr2Parser.processIfds(f, h)
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
		jpegPath, err := gCr2Parser.decodeAndWriteJpeg(f, jpegInfo, testdir, 50)
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
		t.Fatalf("Unable to open test CR2 file: %v\n", e)
	}
}

func TestCr2ProcessFile(t *testing.T) {
	setupCr2()

	// little endian CR2
	testdir, e := getCr2TestDir()
	if e == nil {
		ni := RawFileInfo{TestCR2File, testdir, 50, 1}
		cr2, err := gCr2Parser.ProcessFile(&ni)
		defer os.Remove(cr2.JpegPath)
		if err != nil {
			t.Fatal("Unexpected error while parsing test little endian CR2")
		}
		t.Logf("Parsed little endian CR2: %v\n", cr2)

		// verify jpeg has been extracted
		info, e := os.Stat(cr2.JpegPath)
		if e != nil {
			t.Fail()
		}
		t.Logf("Extracted jpeg details: %v\n", info)
		if info.Size() == 0 {
			t.Fail()
		}
	} else {
		t.Fatal("Unable to determine test directory")
	}

}

func TestCr2ProcessNonExistentFile(t *testing.T) {
	setupCr2()

	testdir, e := getCr2TestDir()
	if e != nil {
		t.Fatal("Unable to determine test directory")
	} else {
		ni := RawFileInfo{"", testdir, 50, 1}
		_, err := gCr2Parser.ProcessFile(&ni)
		if err == nil {
			t.Fatal("Expected error not generated while parsing test little endian CR2")
		} else {
			t.Logf("Received expected error: %v\n", err)
		}
	}
}

func TestEndianessState(t *testing.T) {
	setupCr2()

	if gCr2Parser.SetHostIsLittleEndian(true); gCr2Parser.IsHostLittleEndian() != true {
		t.Fail()
	}

	if gCr2Parser.SetHostIsLittleEndian(false); gCr2Parser.IsHostLittleEndian() != false {
		t.Fail()
	}
}
