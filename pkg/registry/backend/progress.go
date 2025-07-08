package backend

import (
	"io"
	"sync/atomic"
)

type ProgressReader struct {
	io.Reader
	TotalSize    int64
	ReadSize     int64
	ProgressChan chan int64
}

func NewProgressReader(reader io.Reader, total int64, progressChan chan int64) *ProgressReader {
	return &ProgressReader{
		Reader:       reader,
		TotalSize:    total,
		ProgressChan: progressChan,
	}
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	if n > 0 {
		currentRead := atomic.AddInt64(&pr.ReadSize, int64(n))
		// Send current progress to channel with non-blocking approach
		if pr.ProgressChan != nil {
			select {
			case pr.ProgressChan <- currentRead:
				// Progress sent successfully
			default:
				// Channel is full, skip this progress update to prevent blocking
				// This is acceptable as we'll send the next update soon
			}
		}
	}
	// Close progress channel when reading is complete
	if err == io.EOF && pr.ProgressChan != nil {
		close(pr.ProgressChan)
		pr.ProgressChan = nil // Prevent double close
	}
	return
}
