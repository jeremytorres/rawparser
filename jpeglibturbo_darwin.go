// +build turbojpeg

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

// Note: modify these flags for your enviornment if required.

// #cgo CFLAGS: -I/usr/local/opt/jpeg-turbo/include -O2
// #cgo LDFLAGS: -L/usr/local/opt/jpeg-turbo/lib -lturbojpeg
// #include "jpeg_wrapper.h"
import "C"

import (
	"fmt"
	"log"
	"unsafe"
)

func init() {
	log.Println("Using turbojpeg native library")
}

func decodeAndWriteJpeg(data []byte, quality int, filename string) error {
	var rc C.int
	f := C.CString(filename)
	defer C.cleanupString(f)

	rc = C.decodeEncodeWrite((*C.uchar)(unsafe.Pointer(&data[0])),
		C.int(len(data)), C.int(quality), f)

	if rc != 0 {
		return fmt.Errorf("Error re-encoding JPEG")
	}
	return nil
}
