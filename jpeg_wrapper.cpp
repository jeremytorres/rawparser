// +build jpegcpp

#include "jpeg_wrapper.h"
#include "jpgd.h"
#include "jpge.h"

extern "C" void
cleanupString(char *c)
{
    free(c);
}

extern "C" int
decodeEncodeWrite(unsigned char *ci, int ciLen, int quality, char *filename)
{
    static const int requestedComps = 3; // RGB
    int actualComps, w, h;

    // decompress
    unsigned char *decodedBytes = jpgd::decompress_jpeg_image_from_memory(
                                      ci, ciLen, &w, &h, &actualComps, requestedComps);

    if (decodedBytes != NULL) {
        // default params are OK for color images
        jpge::params p;
        p.m_quality = quality;

        // compress using requested quality
        bool result = jpge::compress_image_to_jpeg_file(
                          filename, w, h, actualComps,
                          decodedBytes, p);

        free(decodedBytes);

        if (!result) {
            return 1;
        }
    } else {
        return 1;
    }

    return 0;
}
