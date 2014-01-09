// +build !jpeg,!turbojpeg,!jpegcpp

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
	"bytes"
	"image"
	"image/jpeg"
	"log"
	"os"
)

func init() {
	log.Println("Using pure GO JPEG package")
}

func decodeAndWriteJpeg(data []byte, quality int, filename string) error {
	jpegFile, err := os.Create(filename)
	defer jpegFile.Close()
	if err != nil {
		log.Printf("Error creating jpeg file: %v\n", err)
		return err
	}

	// Decode image
	decodedImage, err := decodeJpeg(data)
	if err != nil {
		log.Printf("Error decoding embedded jpeg: %v\n", err)
		return err
	}

	// Encode and write using specifid JPEG quality
	err = encodeAndWriteJpeg(jpegFile, decodedImage, quality)
	if err != nil {
		log.Printf("Error encoding embedded jpeg: %v\n", err)
	}
	return err
}

func decodeJpeg(data []byte) (img image.Image, e error) {
	// Decode JPEG
	bReader := bytes.NewReader(data)
	img, e = jpeg.Decode(bReader)
	if e != nil {
		log.Printf("Error decoding embedded jpeg: %v\n", e)
		return nil, e
	}
	return img, e
}

// encodeAndWriteJpeg encodes a JPEG image based on a JPEG quality parameter
// from 1 to 100, where 100 is the best encoding quality.
func encodeAndWriteJpeg(f *os.File, img image.Image, q int) error {
	e := jpeg.Encode(f, img, &jpeg.Options{q})
	if e != nil {
		log.Printf("Error encoding and writing embedded jpeg: %v\n", e)
	}
	return e
}
