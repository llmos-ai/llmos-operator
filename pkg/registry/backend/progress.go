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
		atomic.AddInt64(&pr.ReadSize, int64(n))
		// send current progress to channel
		if pr.ProgressChan != nil {
			select {
			case pr.ProgressChan <- pr.ReadSize:
			default:
				// no block
			}
		}
	}
	return
}
