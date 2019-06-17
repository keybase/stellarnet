package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stellar/go/exp/ingest/io"
)

func (p *Pipeline) Node(processor StateProcessor) *PipelineNode {
	return &PipelineNode{
		Processor: processor,
	}
}

func (p *Pipeline) PrintStatus() {
	p.printNodeStatus(p.rootStateProcessor, 0)
}

func (p *Pipeline) printNodeStatus(node *PipelineNode, level int) {
	fmt.Print(strings.Repeat("  ", level))

	var wrRatio = float32(0)
	if node.readEntries > 0 {
		wrRatio = float32(node.wroteEntries) / float32(node.readEntries)
	}

	icon := ""
	if node.queuedEntries > bufferSize/10*9 {
		icon = "⚠️ "
	}

	fmt.Printf(
		"└ %s%s read=%d (queued=%d rps=%d) wrote=%d (w/r ratio = %1.5f) concurrent=%t jobs=%d\n",
		icon,
		node.Processor.Name(),
		node.readEntries,
		node.queuedEntries,
		node.readsPerSecond,
		node.wroteEntries,
		wrRatio,
		node.Processor.IsConcurrent(),
		node.jobs,
	)

	if node.jobs > 1 {
		fmt.Print(strings.Repeat("  ", level))
		fmt.Print("  ")
		for i := 0; i < node.jobs; i++ {
			fmt.Print("• ")
		}
		fmt.Println("")
	}

	for _, child := range node.Children {
		p.printNodeStatus(child, level+1)
	}
}

func (p *Pipeline) AddStateProcessorTree(rootProcessor *PipelineNode) {
	p.rootStateProcessor = rootProcessor
}

func (p *Pipeline) ProcessState(readCloser io.StateReadCloser) <-chan error {
	p.doneMutex.Lock()
	if p.done {
		panic("Pipeline already running or done...")
	}
	p.done = true
	p.doneMutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	return p.processStateNode(ctx, &Store{}, p.rootStateProcessor, readCloser, cancel)
}

func (p *Pipeline) processStateNode(ctx context.Context, store *Store, node *PipelineNode, readCloser io.StateReadCloser, cancel context.CancelFunc) <-chan error {
	outputs := make([]io.StateWriteCloser, len(node.Children))

	for i := range outputs {
		outputs[i] = &bufferedStateReadWriteCloser{}
	}

	var wg sync.WaitGroup

	jobs := 1
	if node.Processor.IsConcurrent() {
		jobs = 20
	}

	node.jobs = jobs

	writeCloser := &multiWriteCloser{
		writers:    outputs,
		closeAfter: jobs,
	}

	errorChan := make(chan error)

	for i := 1; i <= jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := node.Processor.ProcessState(ctx, store, readCloser, writeCloser)
			if err != nil {
				// Protect from cancelling twice and sending multiple errors to err channel
				p.cancelledMutex.Lock()
				defer p.cancelledMutex.Unlock()

				if p.cancelled {
					return
				}
				p.cancelled = true
				cancel()
				errorChan <- err
			}
		}()
	}

	finishUpdatingStats := p.updateStats(node, readCloser, writeCloser)

	for i, child := range node.Children {
		wg.Add(1)
		go func(i int, child *PipelineNode) {
			defer wg.Done()
			done := p.processStateNode(ctx, store, child, outputs[i].(*bufferedStateReadWriteCloser), cancel)
			err := <-done
			if err != nil {
				errorChan <- err
			}
		}(i, child)
	}

	go func() {
		wg.Wait()
		finishUpdatingStats <- true
		select {
		case <-ctx.Done():
			// Do nothing, err already sent to a channel...
		default:
			errorChan <- nil
		}
	}()

	return errorChan
}

func (p *Pipeline) updateStats(node *PipelineNode, readCloser io.StateReadCloser, writeCloser *multiWriteCloser) chan<- bool {
	// Update stats
	interval := time.Second
	done := make(chan bool)
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			// This is not thread-safe: check if Mutex slows it down a lot...
			readBuffer, readBufferIsBufferedStateReadWriteCloser := readCloser.(*bufferedStateReadWriteCloser)

			node.writesPerSecond = (writeCloser.wroteEntries - node.wroteEntries) * int(time.Second/interval)
			node.wroteEntries = writeCloser.wroteEntries

			if readBufferIsBufferedStateReadWriteCloser {
				node.readsPerSecond = (readBuffer.readEntries - node.readEntries) * int(time.Second/interval)
				node.readEntries = readBuffer.readEntries
				node.queuedEntries = readBuffer.QueuedEntries()
			}

			select {
			case <-ticker.C:
				continue
			case <-done:
				// Pipeline done
				return
			}
		}
	}()

	return done
}
