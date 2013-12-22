# RawParser -- A camera RAW file parsing utility
## Overview
RawParser is a GO library for extracting: the embedded JPEGs from a camera RAW file and metadata.  It's current incarnation parses TIFF-based RAW files.  There are existing tools that perform this or similar functionality; however, the reasons for creating this tool:

1. I have many RAW files that are processed using commercial software, yet on occassion, I would like the camera-produced JPEG for comparison.
2. To utilize the concurrency model provided by the [GO](http://golang.org) language to process multiple files without any explicit "traditional" locking (e.g, mutexes)
3. Experiment with GO's "C" package and interfacing with existing C libraries.

My [jpegextract](https://github.com/jeremytorres/jpegextract) utility utilizes this library and may serve as a usage example.

## Dependencies
* GO 1.2 (_maybe_ older GO 1.1.2? but not tested)
* [libjpeg](http://www.ijg.org)
* Optional (highly-recommended):
    * [TurboJpeg](http://www.libjpeg-turbo.org/)
    * If you have many JPEGs to extract, TurboJpeg provides noticebly better performance.
 
## Usage
* Obtain the library:
 
`go get github.com/jeremytorres/rawparser`

* Execute the tests

```bash
cd $GOPATH/src/github.com/jeremytorres/rawparser
```
`go test`


### Current Development Status
- I consider the current status a beta version as there is a laundry list of this I will like to support:
    - Add performance benchmarks
    - Add additional camera RAW file support
    - Create a "better" parser interface

