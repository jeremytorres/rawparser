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
	"log"
	"math"
	"os"
	"time"
)

// Cr2ParserKey is a unique identifier for the CR2 raw file parser.
// This key may be used as a key the RawParsers map.
const Cr2ParserKey = "CR2"

// cr2Header is a struct representing a CR2 file header.
//   Byte Order: offset 0, len 2
//   TIFF Magic Value: offset 2, len 2
//   TIFF Offset Value: offset 4, len 4
//   CR2 Magic Word: offset 8, len 2
//   CR2 Major Version:  offset 10, len 1
//   CR2 Minor Version:  offset 11, len 1
type cr2Header struct {
	isBigEndian                  bool
	tiffMagicValue               uint16
	cr2MagicValue                string
	cr2MajorValue, cr2MinorValue uint8
	tiffOffset                   int64 // offset from start of file
}

// Cr2Parser is the struct defining the state of
// the RawFile concept.  Implements the RawParser interface.
// This parser provides basic parsing functionaity for the Canon Raw Format 2
// (CR2).  For a specified CR2, the EXIF create time and orientation are parsed and the
// embedded JPEG is extracted.  The following are resources on CR2 file details:
//
// CR2-specific information: http://lclevy.free.fr/cr2
// TIFF specification: http://partners.adobe.com/public/developer/en/tiff/TIFF6.pdf
type Cr2Parser struct {
	//HostIsLittleEndian bool
	*rawParser
}

// ProcessFile is the entry point into the Cr2Parser.  For a specified CR2,
// via RawFileInfo, the file shall be processed, JPEG extracted, and
// processed details returned to the caller.
// Returns a pointer the RawFile data structure or error.
func (n Cr2Parser) ProcessFile(info *RawFileInfo) (CR2 *RawFile, err error) {
	CR2 = new(RawFile)

	// file is closed in subsequent method
	f, err := os.Open(info.File)
	if err != nil {
		log.Printf("Error: Unable to open file: '%s'\n", info.File)
	} else {
		h, _ := n.processHeader(f)
		jpegInfo, createDate, err := n.processIfds(f, h)
		if err == nil {
			jpegPath, err := n.decodeAndWriteJpeg(f, jpegInfo, info.DestDir, info.Quality)
			if err == nil {
				CR2.FileName = info.File
				CR2.CreateDate = createDate
				CR2.JpegPath = jpegPath
				CR2.JpegOrientation = jpegInfo.orientation

				log.Printf("========= Processed file %s\n", info.File)
			}
		}
	}

	return CR2, err
}

// processHeader reads CR2 header that defines:
//   byte order;
//   TIFF magic value
//   TIFF offset
// Returns a pointer to the header struct or error.
func (n Cr2Parser) processHeader(f *os.File) (*cr2Header, error) {
	var h cr2Header

	// byte order
	bytes, err := readField(0, 2, f)
	if err != nil {
		return &h, err
	}
	// byte order bytes
	byteOrder := bytesToUShort(n.HostIsLittleEndian, false, bytes)

	// set byte order from header read
	h.isBigEndian = (byteOrder == 0x4D4D)

	// TIFF magic value
	bytes, err = readField(2, 2, f)
	if err != nil {
		return &h, err
	}
	h.tiffMagicValue = bytesToUShort(n.HostIsLittleEndian, h.isBigEndian, bytes)
	//	log.Printf("TIFF Magic Val converted: 0x%x\n", h.tiffMagicValue)

	// TIFF offset
	bytes, err = readField(4, 4, f)
	if err != nil {
		return &h, err
	}
	val := bytesToUInt(n.HostIsLittleEndian, h.isBigEndian, bytes)
	h.tiffOffset = int64(val)
	//	log.Printf("TIFF Offset Val converted: 0x%x\n", h.tiffOffset)

	// cr2 magic val
	bytes, err = readField(8, 2, f)
	if err != nil {
		return &h, err
	}
	// don't convert for endianess for Cr2 magic value
	// Magic Value is 0x4352 "CR"
	h.cr2MagicValue = bytesToASCIIString(bytes)
	//	log.Printf("CR2 Magic Val ASCII converted: %s\n", h.cr2MagicValue)

	// cr2 major num
	bytes, err = readField(10, 1, f)
	if err != nil {
		return &h, err
	}
	h.cr2MajorValue = uint8(bytes[0])
	//	log.Printf("CR2 Major Val converted: 0x%x\n", h.cr2MajorValue)

	// cr2 minor num
	bytes, err = readField(11, 1, f)
	//	log.Printf("CR2 Minor Val converted: 0x%x\n", h.cr2MinorValue)
	if err != nil {
		return &h, err
	}
	h.cr2MinorValue = uint8(bytes[0])

	return &h, err
}

// processIfds reads all currently-supported IFDs from the CR2.  Currently, it parses:
//     jpegInfo - the information pertaining to the embedded jpeg within the CR2;
//     cDate - the EXIF specified CR2 creation time;
//     Note: more EXIF and CR2-specific tags could be parsed in a future release.
// Return jpegInfo, creation date/time or an error.
func (n Cr2Parser) processIfds(f *os.File, h *cr2Header) (j *jpegInfo, cDate time.Time, err error) {
	var jpeg jpegInfo
	offset := h.tiffOffset

	entries, err := processIfd(n.HostIsLittleEndian, h.isBigEndian, offset, f)
	if err != nil {
		return &jpeg, cDate, err
	}

	for e := entries.Front(); e != nil; e = e.Next() {
		entry := e.Value.(ifdEntry)

		switch {
		case entry.tag == 0x0111: // JPEG offset for IFD0
			jpeg.offset = int64(entry.valueOffset)
		case entry.tag == 0x0112: // orientation tag
			o := processShortValue(h.isBigEndian, entry.valueOffset)
			if o == 8 {
				// rotate 270 CW
				rotationRads := 270 * math.Pi / 180
				jpeg.orientation = rotationRads
			} else {
				jpeg.orientation = 0.0
			}
		case entry.tag == 0x0117:
			jpeg.length = int64(entry.valueOffset)
		case entry.tag == 0x011a:
			jpeg.xRes, _, jpeg.xResFloat, err = processRationalEntry(n.HostIsLittleEndian, h.isBigEndian, entry.valueOffset, f)
		case entry.tag == 0x011b:
			jpeg.yRes, _, jpeg.yResFloat, err = processRationalEntry(n.HostIsLittleEndian, h.isBigEndian, entry.valueOffset, f)
		case entry.tag == 0x8769: // EXIF IFD pointer
			// EXIF IFD pointer.  Note: the pointer is the value represented
			// in valueOffset.
			// Read EXIF Entries
			exifEntries, err := processIfd(n.HostIsLittleEndian, h.isBigEndian, int64(entry.valueOffset), f)
			if err != nil {
				return &jpeg, cDate, err
			}

			for exif := exifEntries.Front(); exif != nil; exif = exif.Next() {
				exifEntry := exif.Value.(ifdEntry)
				if exifEntry.tag == 0x9004 {
					createDate, err := processASCIIEntry(&exifEntry, f)
					if err == nil {
						cDate, _ = parseDateTime(createDate)
					}
				}
			}

			// TODO add for future release
			//case entry.tag == 0x010f:
			//mk, _ := processAsciiEntry(&entry, f)
			//log.Printf("Model: %s\n", mk)
			//case entry.tag == 0x0110:
			//model, _ := processAsciiEntry(&entry, f)
			//log.Printf("Model: %s\n", model)

		}
	}

	return &jpeg, cDate, err
}

// decodeAndWriteJpeg extracts the embedded jpeg bytes within a CR2,
// decodes the JPEG data, and then creates a new jpeg file.
// Returns the full path to the jpeg extracted or an error.
func (n Cr2Parser) decodeAndWriteJpeg(f *os.File, j *jpegInfo, destDir string, quality int) (jpegFileName string, err error) {
	// extract jpeg to new file
	jpegFileName = genExtractedJpegName(f, destDir, "_extracted.jpg")
	log.Printf("Creating JPEG file: %s\n", jpegFileName)

	data := make([]byte, j.length)
	_, err = f.ReadAt(data, j.offset)

	if err != nil {
		log.Printf("Error reading embedded jpeg file: %v\n", err)
		return jpegFileName, err
	}

	err = decodeAndWriteJpeg(data, quality, jpegFileName)

	return jpegFileName, err
}

// NewCr2Parser creates an instance of Cr2Parser.
// Returns a pointer to a Cr2Parser instance.
func NewCr2Parser(hostIsLittleEndian bool) (RawParser, string) {
	return &Cr2Parser{&rawParser{hostIsLittleEndian}}, Cr2ParserKey
}
