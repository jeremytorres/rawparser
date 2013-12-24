// +build jpeg turbojpeg

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

#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <jpeglib.h>
#include <setjmp.h>
#include "jpeg_wrapper.h"

const int COLOR_COMPONENTS = 3;

/*
 * ERROR HANDLING:
 *
 * The JPEG library's standard error handler (jerror.c) is divided into
 * several "methods" which you can override individually.  This lets you
 * adjust the behavior without duplicating a lot of code, which you might
 * have to update with each future release.
 *
 * Our example here shows how to override the "error_exit" method so that
 * control is returned to the library's caller when a fatal error occurs,
 * rather than calling exit() as the standard error_exit method does.
 *
 * We use C's setjmp/longjmp facility to return control.  This means that the
 * routine which calls the JPEG library must first execute a setjmp() call to
 * establish the return point.  We want the replacement error_exit to do a
 * longjmp().  But we need to make the setjmp buffer accessible to the
 * error_exit routine.  To do this, we make a private extension of the
 * standard JPEG error handler object.  (If we were using C++, we'd say we
 * were making a subclass of the regular error handler.)
 *
 * Here's the extended error handler struct:
 */

struct my_error_mgr
{
    struct jpeg_error_mgr pub;    /* "public" fields */

    jmp_buf setjmp_buffer;    /* for return to caller */
};

typedef struct my_error_mgr *my_error_ptr;

/*
 * Here's the routine that will replace the standard error_exit method:
 */

METHODDEF(void)
my_error_exit (j_common_ptr cinfo)
{
    /* cinfo->err really points to a my_error_mgr struct, so coerce pointer */
    my_error_ptr myerr = (my_error_ptr) cinfo->err;

    /* Always display the message. */
    /* We could postpone this until after returning, if we chose. */
    (*cinfo->err->output_message) (cinfo);

    /* Return control to the setjmp point */
    longjmp(myerr->setjmp_buffer, 1);
}

// Derived from turbojpeg documenation and
//  http://blog.thelifeofkenneth.com/2012/07/decompressing-jpegs-in-ram.html
int decompressJpeg(unsigned char *data, size_t len, unsigned char **buf, unsigned long *bufSize, int *width, int *height)
{
    int row_stride, pixel_size;

    struct jpeg_decompress_struct cinfo;
    //    struct jpeg_error_mgr jerr;

    /* We use our private extension JPEG error handler.
    * Note that this struct must live as long as the main JPEG parameter
    * struct, to avoid dangling-pointer problems.
    */
    struct my_error_mgr jerr;

    /* We set up the normal JPEG error routines, then override error_exit. */
    cinfo.err = jpeg_std_error(&jerr.pub);
    jerr.pub.error_exit = my_error_exit;
    /* Establish the setjmp return context for my_error_exit to use. */
    if (setjmp(jerr.setjmp_buffer))
    {
        /* If we get here, the JPEG code has signaled an error.
         * We need to clean up the JPEG object, close the input file, and return.
         */
        jpeg_destroy_decompress(&cinfo);

        *buf = NULL;
        *bufSize = 0;
        *width = -1;
        *height = -1;

        return 1;
    }

    // Setup decompression structure
    cinfo.err = jpeg_std_error((struct jpeg_error_mgr *)&jerr);

    jpeg_create_decompress(&cinfo);

    jpeg_mem_src(&cinfo, data, len);

    // read info from header.
    (void) jpeg_read_header(&cinfo, TRUE);
    jpeg_start_decompress(&cinfo);


    *width = cinfo.output_width;
    *height = cinfo.output_height;
    pixel_size = cinfo.output_components;

    *bufSize = cinfo.output_width * cinfo.output_height * pixel_size;
    *buf = (unsigned char *) malloc(*bufSize);

    /* We may need to do some setup of our own at this point before reading
    * the data.  After jpeg_start_decompress() we have the correct scaled
    * output image dimensions available, as well as the output colormap
    * if we asked for color quantization.
    * In this example, we need to make an output work buffer of the right size.
    */
    /* JSAMPLEs per row in output buffer */
    row_stride = cinfo.output_width * cinfo.output_components;
    /* Make a one-row-high sample array that will go away when done with image */
    //buffer = (*cinfo.mem->alloc_sarray)
    //         ((j_common_ptr) &cinfo, JPOOL_IMAGE, row_stride, 1);

    /* Step 6: while (scan lines remain to be read) */
    /*           jpeg_read_scanlines(...); */

    /* Here we use the library's state variable cinfo.output_scanline as the
     * loop counter, so that we don't have to keep track ourselves.
     */
    while (cinfo.output_scanline < cinfo.output_height)
    {
        /* jpeg_read_scanlines expects an array of pointers to scanlines.
         * Here the array is only one element long, but you could ask for
         * more than one scanline at a time if that's more convenient.
         */
        //(void) jpeg_read_scanlines(&cinfo, *buf, 1);
        /* Assume put_scanline_someplace wants a pointer and sample count. */
        //put_scanline_someplace(*buf[0], row_stride);

        unsigned char *buffer_array[1];
        buffer_array[0] = *buf + (cinfo.output_scanline) * row_stride;

        jpeg_read_scanlines(&cinfo, buffer_array, 1);
    }

    /* Step 7: Finish decompression */

    (void) jpeg_finish_decompress(&cinfo);
    /* We can ignore the return value since suspension is not possible
     * with the stdio data source.
     */

    jpeg_destroy_decompress(&cinfo);

    return 0;
}

inline int
decodeEncodeWrite(unsigned char *ci, int ciLen, int quality, char *filename)
{

    // Pointer to decoded jpeg image.  Note: this function MUST free when
    // no longer needed.
    unsigned char *buffer;
    unsigned long bufLen;
    int rc, w, h;


    rc = decompressJpeg(ci, ciLen, &buffer, &bufLen, &w, &h);

    if (rc != 0)
    {
        return 1;
    }

    /* This struct contains the JPEG compression parameters and pointers to
    * working space (which is allocated as needed by the JPEG library).
    * It is possible to have several such structures, representing multiple
    * compression/decompression processes, in existence at once.  We refer
    * to any one struct (and its associated working data) as a "JPEG object".
    */
    struct jpeg_compress_struct cinfo;
    /* This struct represents a JPEG error handler.  It is declared separately
     * because applications often want to supply a specialized error handler
     * (see the second half of this file for an example).  But here we just
     * take the easy way out and use the standard error handler, which will
     * print a message on stderr and call exit() if compression fails.
     * Note that this struct must live as long as the main JPEG parameter
     * struct, to avoid dangling-pointer problems.
     */
    struct jpeg_error_mgr jerr;
    /* More stuff */
    FILE *outfile;                 /* target file */
    JSAMPROW row_pointer[1];        /* pointer to JSAMPLE row[s] */
    int row_stride;                /* physical row width in image buffer */

    /* Step 1: allocate and initialize JPEG compression object */

    /* We have to set up the error handler first, in case the initialization
     * step fails.  (Unlikely, but it could happen if you are out of memory.)
     * This routine fills in the contents of struct jerr, and returns jerr's
     * address which we place into the link field in cinfo.
     */
    cinfo.err = jpeg_std_error(&jerr);
    /* Now we can initialize the JPEG compression object. */
    jpeg_create_compress(&cinfo);

    /* Step 2: specify data destination (eg, a file) */
    /* Note: steps 2 and 3 can be done in either order. */

    /* Here we use the library-supplied code to send compressed data to a
     * stdio stream.  You can also write your own code to do something else.
     * VERY IMPORTANT: use "b" option to fopen() if you are on a machine that
     * requires it in order to write binary files.
     */
    if ((outfile = fopen(filename, "wb")) == NULL)
    {
        fprintf(stderr, "can't open %s\n", filename);
        return 1;
    }
    jpeg_stdio_dest(&cinfo, outfile);

    /* Step 3: set parameters for compression */

    /* First we supply a description of the input image.
     * Four fields of the cinfo struct must be filled in:
     */
    cinfo.image_width = w;         /* image width and height, in pixels */
    cinfo.image_height = h;
    cinfo.input_components = 3;                /* # of color components per pixel */
    cinfo.in_color_space = JCS_RGB;         /* colorspace of input image */
    /* Now use the library's routine to set default compression parameters.
     * (You must set at least cinfo.in_color_space before calling this,
     * since the defaults depend on the source color space.)
     */
    jpeg_set_defaults(&cinfo);
    /* Now you can set any non-default parameters you wish to.
     * Here we just illustrate the use of quality (quantization table) scaling:
     */
    jpeg_set_quality(&cinfo, quality, TRUE /* limit to baseline-JPEG values */);

    /* Step 4: Start compressor */

    /* TRUE ensures that we will write a complete interchange-JPEG file.
     * Pass TRUE unless you are very sure of what you're doing.
     */
    jpeg_start_compress(&cinfo, TRUE);

    /* Step 5: while (scan lines remain to be written) */
    /*           jpeg_write_scanlines(...); */

    /* Here we use the library's state variable cinfo.next_scanline as the
     * loop counter, so that we don't have to keep track ourselves.
     * To keep things simple, we pass one scanline per call; you can pass
     * more if you wish, though.
     */
    row_stride = w * 3;        /* JSAMPLEs per row in image_buffer */

    while (cinfo.next_scanline < cinfo.image_height)
    {
        /* jpeg_write_scanlines expects an array of pointers to scanlines.
         * Here the array is only one element long, but you could pass
         * more than one scanline at a time if that's more convenient.
         */
        row_pointer[0] = &buffer[cinfo.next_scanline * row_stride];
        (void) jpeg_write_scanlines(&cinfo, row_pointer, 1);
    }

    /* Step 6: Finish compression */

    jpeg_finish_compress(&cinfo);
    /* After finish_compress, we can close the output file. */
    fclose(outfile);

    // free buffer allocated
    free(buffer);

    /* Step 7: release JPEG compression object */

    /* This is an important step since it will release a good deal of memory. */
    jpeg_destroy_compress(&cinfo);

    /* And we're done! */

    return 0;
}

void
cleanupString(char *c)
{
    free(c);
}
