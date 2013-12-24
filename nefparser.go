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
	"fmt"
	"log"
	"math"
	"os"
	"time"
)

// NefParserKey is a unique identifier for the NEF raw file parser.
// This key may be used as a key the RawParsers map.
const NefParserKey = "NEF"

// nefHeader is a struct representing a NEF file header.
//   Byte Order: offset 0, len 2
//   TIFF Magic Value: offset 2, len 2
//   TIFF Offset Value: offset 4, len 4
type nefHeader struct {
	isBigEndian    bool
	tiffMagicValue uint16
	tiffOffset     int64 // offset from start of file
}

// NefParser is the struct defining the state of
// the RawFile concept.  Implements the RawParser interface.
// This parser provides basic parsing functionaity for the Nikon Electronic Format
// (NEF).  For a specified NEF, the EXIF create time and orientation are parsed and the
// embedded JPEG is extracted.  The following are resources on NEF file details:
//
// NEF-specific information: http://lclevy.free.fr/nef/
// TIFF specification: http://partners.adobe.com/public/developer/en/tiff/TIFF6.pdf
type NefParser struct {
	HostIsLittleEndian bool
}

// ProcessFile is the entry point into the NefParser.  For a specified NEF,
// via RawFileInfo, the file shall be processed, JPEG extracted, and
// processed details returned to the caller.
// Returns a pointer the RawFile data structure or error.
func (n NefParser) ProcessFile(info *RawFileInfo) (nef *RawFile, err error) {
	nef = new(RawFile)

	// file is closed in subsequent method
	f, err := os.Open(info.File)
	if err != nil {
		log.Printf("Error: Unable to open file: '%s'\n", info.File)
	} else {
		h, err := n.processHeader(f)
		jpegInfo, createDate, err := n.processIfds(f, h)
		if err != nil {
			return nef, err
		} else if jpegInfo.length <= 0 {
			return nef, fmt.Errorf("invalid jpeg length: %d\n", jpegInfo.length)
		}
		jpegPath, err := n.decodeAndWriteJpeg(f, jpegInfo, info.DestDir, info.Quality)
		if err == nil {
			nef.FileName = info.File
			nef.CreateDate = createDate
			nef.JpegPath = jpegPath
			nef.JpegOrientation = jpegInfo.orientation

			log.Printf("========= Processed file %s\n", info.File)
		}

	}

	return nef, err
}

// SetHostIsLittleEndian is a function to set the host's
// endianness for the given instance of the NefParser.
// Set to true if host is a little endian machine; false otherwise.
func (n *NefParser) SetHostIsLittleEndian(hostIsLe bool) {
	n.HostIsLittleEndian = hostIsLe
}

// IsHostLittleEndian is a function to get the host's
// endianness specified for the given instance of the NefParser.
// Returns true if the host is a little endian machine.
func (n NefParser) IsHostLittleEndian() bool {
	return n.HostIsLittleEndian
}

// processHeader reads NEF header that defines:
//   byte order;
//   TIFF magic value
//   TIFF offset
// Returns a pointer to the header struct or error.
func (n NefParser) processHeader(f *os.File) (*nefHeader, error) {
	var h nefHeader

	// byte order
	bytes, err := readField(0, 2, f)
	if err != nil {
		return &h, err
	}
	// byte order
	byteOrder := bytesToUShort(n.IsHostLittleEndian(), false, bytes)

	// set byte order from file read
	h.isBigEndian = (byteOrder == 0x4D4D)

	// DEBUG
	//if !h.isBigEndian {
	//log.Println("NEF is LITTLE ENDIAN!")
	//}
	// DEBUG

	// TIFF magic value
	bytes, err = readField(2, 2, f)
	if err != nil {
		return &h, err
	}
	h.tiffMagicValue = bytesToUShort(n.IsHostLittleEndian(), h.isBigEndian, bytes)

	// TIFF offset
	bytes, err = readField(4, 4, f)
	if err != nil {
		return &h, err
	}
	val := bytesToUInt(n.IsHostLittleEndian(), h.isBigEndian, bytes)
	h.tiffOffset = int64(val)

	return &h, err
}

// processIfds reads all currently-supported IFDs from the NEF.  Currently, it parses:
//     jpegInfo - the information pertaining to the embedded jpeg within the NEF;
//     cDate - the EXIF specified NEF creation time;
//     Note: more EXIF and NEF-specific tags could be parsed in a future release.
// Return jpegInfo, creation date/time or an error.
func (n NefParser) processIfds(f *os.File, h *nefHeader) (j *jpegInfo, cDate time.Time, err error) {
	var jpeg jpegInfo
	offset := h.tiffOffset

	entries, err := processIfd(n.IsHostLittleEndian(), h.isBigEndian, offset, f)

	if err == nil {
		for e := entries.Front(); e != nil; e = e.Next() {
			entry := e.Value.(ifdEntry)
			if entry.tag == 0x014a { // SUBID
				// JPEG offset (SUBID 0)
				bytes, err := readField(int64(entry.valueOffset), 4, f)
				if err == nil {
					subID0Offset := int64(bytesToUInt(n.IsHostLittleEndian(), h.isBigEndian, bytes))

					// Read SUBIFD 0 for JPEG
					subIfd0Entries, err := processIfd(n.IsHostLittleEndian(), h.isBigEndian, subID0Offset, f)
					if err == nil {
						for se := subIfd0Entries.Front(); se != nil; se = se.Next() {
							subID0Entry := se.Value.(ifdEntry)

							if subID0Entry.tag == 0x011a {
								jpeg.xRes, _, jpeg.xResFloat, err = processRationalEntry(n.IsHostLittleEndian(), h.isBigEndian, subID0Entry.valueOffset, f)
							}

							if subID0Entry.tag == 0x011b {
								jpeg.yRes, _, jpeg.yResFloat, err = processRationalEntry(n.IsHostLittleEndian(), h.isBigEndian, subID0Entry.valueOffset, f)
							}

							if subID0Entry.tag == 0x0201 {
								jpeg.offset = int64(subID0Entry.valueOffset)
							}
							if subID0Entry.tag == 0x0202 {
								jpeg.length = int64(subID0Entry.valueOffset)
							}
						}
					} else {
						return &jpeg, cDate, err
					}
				}
			} else if entry.tag == 0x0112 { // orientation tag
				o := processShortValue(h.isBigEndian, entry.valueOffset)
				if o == 8 {
					// rotate 270 CW
					rotationRads := 270 * math.Pi / 180
					jpeg.orientation = rotationRads
				} else {
					jpeg.orientation = 0.0
				}
			} else if entry.tag == 0x8769 { // EXIF IFD pointer
				// EXIF IFD pointer.  Note: the pointer is the value represented
				// in valueOffset.

				// Read EXIF Entries
				exifEntries, err := processIfd(n.IsHostLittleEndian(), h.isBigEndian, int64(entry.valueOffset), f)
				if err == nil {
					for exif := exifEntries.Front(); exif != nil; exif = exif.Next() {
						exifEntry := exif.Value.(ifdEntry)
						if exifEntry.tag == 0x9004 {
							createDate, err := processASCIIEntry(&exifEntry, f)
							if err == nil {
								cDate, err = parseDateTime(createDate)
							}
						}
					}
				} else {
					return &jpeg, cDate, err
				}
			}
		}
	}

	return &jpeg, cDate, err
}

// decodeAndWriteJpeg extracts the embedded jpeg bytes within a NEF,
// decodes the JPEG data, and then creates a new jpeg file.
// Returns the full path to the jpeg extracted or an error.
func (n NefParser) decodeAndWriteJpeg(f *os.File, j *jpegInfo, destDir string, quality int) (jpegFileName string, err error) {
	// extract jpeg to new file
	jpegFileName = genExtractedJpegName(f, destDir, "_extracted.jpg")
	log.Printf("Creating JPEG file: %s\n", jpegFileName)

	/* Uncomment this block if you want to use GO's image/jpeg facility
	jpegFile, err := os.Create(jpegFileName)

	defer jpegFile.Close()

	if err != nil {
		log.Printf("Error creating jpeg file: %v\n", err)
		return jpegFileName, err
	}

	// Decode image
	decodedImage, err := decodeJpeg(f, j)
	if err != nil {
		log.Printf("Error decoding embedded jpeg: %v\n", err)
		return jpegFileName, err
	}

	// Encode and write using specifid JPEG quality
	err = encodeAndWriteJpeg(jpegFile, decodedImage, quality)
	if err != nil {
		log.Printf("Error encoding embedded jpeg: %v\n", err)
	}
	*/

	data := make([]byte, j.length)
	_, err = f.ReadAt(data, j.offset)

	if err != nil {
		log.Printf("Error reading embedded jpeg file: %v\n", err)
		return jpegFileName, err
	}

	err = decodeAndWriteJpeg(data, quality, jpegFileName)

	return jpegFileName, err
}

// NewNefParser creates an instance of NEF-specific RawParser.
// Returns an instance of a NEF-specific RawParser.
func NewNefParser(hostIsLittleEndian bool) (RawParser, string) {
	return &NefParser{hostIsLittleEndian}, NefParserKey
}
