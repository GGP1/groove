package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const (
	// defaultMinSize is the default minimum size until we enable gzip compression.
	// 1500 bytes is the MTU size for the internet since that is the largest size allowed at the network layer.
	// If you take a file that is 1300 bytes and compress it to 800 bytes, it’s still transmitted in that same 1500 byte packet regardless, so you’ve gained nothing.
	// That being the case, you should restrict the gzip compression to files with a size (plus header) greater than a single packet,
	// 1024 bytes (1KB) is therefore default.
	defaultMinSize = 1024

	acceptEncoding  = "Accept-Encoding"
	contentEncoding = "Content-Encoding"
	contentLength   = "Content-Length"
)

var gzipPool = sync.Pool{
	New: func() any {
		return newGzipResponseWriter(nil)
	},
}

// GzipCompress checks if the request accepts encoding and utilized gzip or proceed without compressing.
func GzipCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clHeader := r.Header[contentLength]
		if len(clHeader) != 0 {
			cl, _ := atoi(clHeader[0])
			if cl < defaultMinSize {
				next.ServeHTTP(w, r)
				return
			}
		}

		if !acceptsGzip(r.Header[acceptEncoding][0]) {
			next.ServeHTTP(w, r)
			return
		}

		gw := gzipPool.Get().(*GzipReponseWriter)
		// Reset writer to the current ResponseWriter, there is no need to Flush
		gw.Reset(w)
		gw.Header()[contentEncoding] = []string{"gzip"}

		next.ServeHTTP(gw, r)

		gw.Close()
		gzipPool.Put(gw)
	})
}

// GzipReponseWriter is a response writer containing a GZIP writer in it
type GzipReponseWriter struct {
	w  http.ResponseWriter
	gw *gzip.Writer
}

// newGzipResponseWriter returns a new GZIPResponseWriter.
func newGzipResponseWriter(rw http.ResponseWriter) *GzipReponseWriter {
	return &GzipReponseWriter{w: rw, gw: gzip.NewWriter(io.Discard)}
}

// Close closes the gzip writer.
func (g *GzipReponseWriter) Close() {
	g.gw.Close()
}

// Header is implemented to satisfy the response writer interface.
func (g *GzipReponseWriter) Header() http.Header {
	return g.w.Header()
}

// Reset sets a new writer for the gzip and updates the struct writer.
func (g *GzipReponseWriter) Reset(w http.ResponseWriter) {
	g.w = w
	g.gw.Reset(w)
}

// Write is implemented to satisfy the response writer interface.
func (g *GzipReponseWriter) Write(d []byte) (int, error) {
	return g.gw.Write(d)
}

// WriteHeader is implemented to satisfy the response writer interface.
func (g *GzipReponseWriter) WriteHeader(statuscode int) {
	g.w.WriteHeader(statuscode)
}

// acceptsGzip returns whether the client will accept gzip-encoded content.
func acceptsGzip(key string) bool {
	parts := strings.Split(key, ",")
	for _, part := range parts {
		if strings.HasPrefix(strings.TrimSpace(part), "gzip") {
			return true
		}
	}
	return false
}

const intSize = 32 << (^uint(0) >> 63)

// atoi is equivalent to ParseInt(s, 10, 0), converted to type int.
func atoi(s string) (int, bool) {
	sLen := len(s)
	if intSize == 32 && (0 < sLen && sLen < 10) ||
		intSize == 64 && (0 < sLen && sLen < 19) {
		// Fast path for small integers that fit int type.
		s0 := s
		if s[0] == '-' || s[0] == '+' {
			s = s[1:]
			if len(s) < 1 {
				return 0, false
			}
		}

		n := 0
		for _, ch := range []byte(s) {
			ch -= '0'
			if ch > 9 {
				return 0, false
			}
			n = n*10 + int(ch)
		}
		if s0[0] == '-' {
			n = -n
		}
		return n, true
	}

	// Slow path for invalid, big, or underscored integers.
	i64, err := strconv.ParseInt(s, 10, 0)
	return int(i64), err == nil
}
