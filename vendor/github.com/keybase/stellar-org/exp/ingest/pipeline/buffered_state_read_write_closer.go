package pipeline

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
)

const bufferSize = 50000

func (b *bufferedStateReadWriteCloser) init() {
	b.buffer = make(chan xdr.LedgerEntryChange, bufferSize)
}

func (b *bufferedStateReadWriteCloser) close() {
	b.writeCloseMutex.Lock()
	defer b.writeCloseMutex.Unlock()

	close(b.buffer)
	b.closed = true
}

func (b *bufferedStateReadWriteCloser) GetSequence() uint32 {
	return 0
}

func (b *bufferedStateReadWriteCloser) Read() (xdr.LedgerEntryChange, error) {
	b.initOnce.Do(b.init)

	entry, more := <-b.buffer
	if more {
		b.readEntriesMutex.Lock()
		b.readEntries++
		b.readEntriesMutex.Unlock()
		return entry, nil
	} else {
		return xdr.LedgerEntryChange{}, io.EOF
	}
}

func (b *bufferedStateReadWriteCloser) Write(entry xdr.LedgerEntryChange) error {
	b.initOnce.Do(b.init)

	b.writeCloseMutex.Lock()
	defer b.writeCloseMutex.Unlock()

	if b.closed {
		return io.ErrClosedPipe
	}

	b.buffer <- entry
	b.wroteEntries++
	return nil
}

func (b *bufferedStateReadWriteCloser) QueuedEntries() int {
	b.initOnce.Do(b.init)
	return len(b.buffer)
}

// Close can be called in `StateWriteCloser` and `StateReadCloser` context.
//
// In `StateReadCloser` it means that no more values will be read so writer can
// stop writing to a buffer (`io.ErrClosedPipe` will be returned for calls to
// `Write()`).
//
// In `StateWriteCloser` it means that no more values will be written so reader
// should start returning `io.EOF` error after returning all queued values.
func (b *bufferedStateReadWriteCloser) Close() error {
	b.initOnce.Do(b.init)
	b.closeOnce.Do(b.close)
	return nil
}

var _ io.StateReadCloser = &bufferedStateReadWriteCloser{}
var _ io.StateWriteCloser = &bufferedStateReadWriteCloser{}
