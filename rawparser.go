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

// Package rawparser provides a basic parsing interface for camera raw files.  The current
// incarnation supports TIFF-based RAW files (e.g., Canon CR2, Nikon NEF...).
//
// TIFF specification: http://partners.adobe.com/public/developer/en/tiff/TIFF6.pdf
package rawparser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ifdEntry is a struct representing a TIFF Image File Directory (IFD).
// Each 12-byte IFD entry has the following format:
//   Bytes 0-1 The Tag that identifies the field.
//   Bytes 2-3 The field Type.
//   Bytes 4-7 The number of values, Count of the indicated Type.
//   Bytes 8-11 The Value Offset, the file offset (in bytes) of the Value for the field.
type ifdEntry struct {
	tag, fieldType     uint16
	count, valueOffset uint32 // offset from start of file
}

// jpegInfo is a struct representing a RawFile'sembedded jpeg information.
type jpegInfo struct {
	orientation          float64
	offset, length       int64
	xRes, yRes           uint32
	xResFloat, yResFloat float64
}

// RawFileInfo is a struct defining key information for parsing a RawFile.
type RawFileInfo struct {
	File          string
	DestDir       string
	Quality       int
	NumOfChannels int
}

// RawFile is a struct representing parsed results for a specific raw file.
type RawFile struct {
	// Note: additional EXIF metadata may be added in future release.
	CreateDate         time.Time
	FileName, JpegPath string
	JpegOrientation    float64
}

// RawParser is the defining interface of a raw file parser.  Camera-specific parsers
// shall implement this interface.
type RawParser interface {
	// ProcessFile processes a raw file per the implementation of this parser.
	// Return a pointer to a RawFile struct or error.
	ProcessFile(i *RawFileInfo) (r *RawFile, e error)

	// SetHostIsLittleEndian is a function to set the RawParser host's
	// endianness.
	// Set to true if host is a little endian machine; false otherwise.
	SetHostIsLittleEndian(b bool)

	// IsLittleEndian is a function to get the value of the specified host
	// endianness.
	// Returns true if the host is a little endian machine.
	IsHostLittleEndian() bool
}

// RawParsers is a structure containing a mapping
// of registered raw file parsers.  The key is the
// lower-case file extension of the raw file type;
// the value is the pointer to the RawParser implementation.
type RawParsers struct {
	parserMap map[string]RawParser
}

// NewRawParsers creates an instance of RawParsers.
func NewRawParsers() *RawParsers {
	p := new(RawParsers)
	p.parserMap = make(map[string]RawParser)
	return p
}

// Register maps the implementation of the RawParser
// interface to the key.
func (p *RawParsers) Register(key string, parser RawParser) {
	p.parserMap[key] = parser
}

// GetParser returns a RawParser for a given raw file type or nil if not found.
func (p RawParsers) GetParser(key string) RawParser {
	return p.parserMap[key]
}

// DeleteParser removes the specified RawParser.
func (p *RawParsers) DeleteParser(key string) {
	delete(p.parserMap, key)
}

// parseDateTime converts a TIFF-based date/time string into a time.Time.
// Returns a time.Time or error.
func parseDateTime(s string) (t time.Time, err error) {
	const format = "02 Jan 06 15:04"

	split := strings.Split(s, " ")
	if len(split) != 2 {
		return t, fmt.Errorf("dateTime string invalid: '%s'", s)
	}

	dateToken := split[0]
	timeToken := split[1]
	dateTokens := strings.Split(dateToken, ":")
	timeTokens := strings.Split(timeToken, ":")

	if len(dateTokens) == 3 && len(timeTokens) == 3 {
		montStr, err := toRfc822Date(dateTokens)
		if err != nil {
			return t, err
		}
		dateStr := dateTokens[2] + " " + montStr + " " + string(dateTokens[0][2]) + string(dateTokens[0][3])
		t, err = time.Parse(format, dateStr+" "+timeTokens[0]+":"+timeTokens[1])
		if err != nil {
			return t, err
		}
	} else {
		err = fmt.Errorf("invalid date and/or time string format: %s %s\n", dateToken, timeToken)
	}

	return t, err
}

// toRfc822Date converts a TIFF-based numerical month to an RFC822, 3-digit
// alpha date.
// Returns 3-digit date string or error.
func toRfc822Date(dateTokens []string) (string, error) {
	var monthStr string
	var e error

	token := dateTokens[1]

	// numerical date to 3-digit alpa
	switch token {
	case "01":
		monthStr = "Jan"
	case "02":
		monthStr = "Feb"
	case "03":
		monthStr = "Mar"
	case "04":
		monthStr = "Apr"
	case "05":
		monthStr = "May"
	case "06":
		monthStr = "Jun"
	case "07":
		monthStr = "Jul"
	case "08":
		monthStr = "Aug"
	case "09":
		monthStr = "Sep"
	case "10":
		monthStr = "Oct"
	case "11":
		monthStr = "Nov"
	case "12":
		monthStr = "Dec"
	default:
		e = fmt.Errorf("invalid month: '%s'\n", token)
	}

	return monthStr, e
}

// genExtractedJpegName creates a full path name for an extracted JPEG
// from a raw file.
// The input file is the pointer to the raw file and its base name is used
// as the base of the JPEG files; destDir is the full path
// to the destination directory containing the JPEG file; and suffix is
// the remainder of the file name including file extension.
// Example:
//     destDir="/path_to/outputDir"
//     suffix="_extracted.jpg"
// Returns fully-qualified path to the JPEG extraced from the raw file.
func genExtractedJpegName(f *os.File, destDir, suffix string) string {
	return destDir + filepath.Base(f.Name()) + suffix
}
