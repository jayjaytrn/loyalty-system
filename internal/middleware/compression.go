package middleware

import (
	"compress/gzip"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

type (
	gzipWriter struct {
		http.ResponseWriter
		GzipWriter io.Writer
	}

	gzipReader struct {
		r          io.ReadCloser
		GzipReader *gzip.Reader
	}

	Middleware func(http.Handler, *zap.SugaredLogger) http.Handler
)

func WriteWithCompression(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "text/html" {
			sugar.Info("Content-Type is not supported for compression. Content-Type: " + contentType)
			h.ServeHTTP(w, r)
			return
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if !supportsGzip {
			sugar.Info("Accept-Encoding is not allowed")
			h.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			sugar.Error("Failed to create gzip writer", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		h.ServeHTTP(gzipWriter{ResponseWriter: w, GzipWriter: gz}, r)
	})
}

// TODO прочитать как будто это не нужно тут и так
func (w gzipWriter) Write(b []byte) (int, error) {
	return w.GzipWriter.Write(b)
}

func ReadWithCompression(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if !sendsGzip {
			sugar.Info("Content-Encoding is not allowed")
			h.ServeHTTP(w, r)
			return
		}

		gz, err := newGzipReader(r.Body)
		if err != nil {
			sugar.Error("Failed to create gzip reader", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		r.Body = gz
		defer gz.Close()
		defer r.Body.Close()

		h.ServeHTTP(w, r)
	})
}

func (r *gzipReader) Read(p []byte) (n int, err error) {
	return r.GzipReader.Read(p)
}

func (r *gzipReader) Close() error {
	if err := r.r.Close(); err != nil {
		return err
	}
	return r.GzipReader.Close()
}

func newGzipReader(r io.ReadCloser) (*gzipReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &gzipReader{
		r:          r,
		GzipReader: zr,
	}, nil
}
