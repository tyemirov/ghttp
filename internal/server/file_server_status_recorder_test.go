package server

import (
	"net/http"
	"testing"
)

func TestStatusRecorderFlushPassesThroughToWrappedWriter(t *testing.T) {
	trackingWriter := &flushTrackingResponseWriter{header: http.Header{}}
	recorder := newStatusRecorder(trackingWriter)

	responseFlusher, supportsFlush := interface{}(recorder).(http.Flusher)
	if !supportsFlush {
		t.Fatalf("expected status recorder to implement http.Flusher")
	}
	responseFlusher.Flush()
	if trackingWriter.flushCount != 1 {
		t.Fatalf("expected wrapped writer flush to be called once, got %d", trackingWriter.flushCount)
	}
	if recorder.Unwrap() != trackingWriter {
		t.Fatalf("expected unwrap to return original writer")
	}
}

type flushTrackingResponseWriter struct {
	header     http.Header
	flushCount int
}

func (writer *flushTrackingResponseWriter) Header() http.Header {
	return writer.header
}

func (writer *flushTrackingResponseWriter) Write(content []byte) (int, error) {
	return len(content), nil
}

func (writer *flushTrackingResponseWriter) WriteHeader(_ int) {
}

func (writer *flushTrackingResponseWriter) Flush() {
	writer.flushCount++
}
