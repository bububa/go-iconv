// Bindings for iconv. Iconv is a set of functions used to convert strings
// between different character sets
package iconv

// #cgo LDFLAGS: -liconv
// #include <stdlib.h>
// #include <iconv.h>
// #include <errno.h>
/*
// Used to fix GO1.6 cgo panic: cgo argument has Go pointer to Go pointer.
size_t iconv_custom(iconv_t cd,
                    char *inbuf, size_t *inbytesleft,
                    char *outbuf, size_t *outbytesleft) {
	return iconv(cd, &inbuf, inbytesleft, &outbuf, outbytesleft);
}
*/
import "C"

import (
	"bytes"
	"syscall"
	"unsafe"
)

const bufSize = 512

// Opaque structure containing the internal state of the codec
type Iconv struct {
	pointer C.iconv_t
}

// Create a codec which convert a string encoded in fromcode into a string
// encoded in tocode
//
// If you add //TRANSLIT at the end of tocode, any character which doesn't
// exists in the destination charset will be replaced by its closest
// equivalent (for example, € will be represented by EUR in ASCII). Else,
// such a character will trigger an error.
func Open(tocode string, fromcode string) (*Iconv, error) {
	tocodeC := C.CString(tocode)
	fromcodeC := C.CString(fromcode)
	defer func() {
		// free the C Strings
		C.free(unsafe.Pointer(tocodeC))
		C.free(unsafe.Pointer(fromcodeC))
	}()

	ret, err := C.iconv_open(tocodeC, fromcodeC)
	if err != nil {
		return nil, err
	}
	return &Iconv{ret}, nil
}

// Destroy the internal state of the codec, releasing associated memory
func (cd *Iconv) Close() error {
	_, err := C.iconv_close(cd.pointer)
	return err
}

// Use the codec to convert a string
func (cd *Iconv) Conv(input string) (result string, err error) {
	var buf bytes.Buffer

	if len(input) == 0 {
		return "", nil
	}

	inbuf := []byte(input)
	outbuf := make([]byte, bufSize)
	inbytes := C.size_t(len(inbuf))
	inptr := &inbuf[0]

	for inbytes > 0 {
		outbytes := C.size_t(len(outbuf))
		outptr := &outbuf[0]
		inbytes_orig := inbytes
		_, err = C.iconv_custom(cd.pointer,
			(*C.char)(unsafe.Pointer(inptr)), &inbytes,
			(*C.char)(unsafe.Pointer(outptr)), &outbytes)
		buf.Write(outbuf[:len(outbuf)-int(outbytes)])
		if err != nil && err != syscall.E2BIG {
			return buf.String(), err
		}

		if inbytes > 0 {
			inptr = &inbuf[inbytes_orig-inbytes]
		}
	}

	return buf.String(), nil
}

// Utility function which create a codec, convert the string and then destroy it
func Conv(input string, tocode string, fromcode string) (string, error) {
	h, err := Open(tocode, fromcode)
	if err != nil {
		return "", err
	}
	defer h.Close()
	return h.Conv(input)
}
