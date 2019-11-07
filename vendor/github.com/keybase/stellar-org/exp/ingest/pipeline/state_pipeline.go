package pipeline

import (
	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
)

func StateNode(processor StateProcessor) *supportPipeline.PipelineNode {
	return &supportPipeline.PipelineNode{
		Processor: &stateProcessorWrapper{processor},
	}
}

func (p *StatePipeline) Process(reader io.StateReader) <-chan error {
	return p.Pipeline.Process(&stateReaderWrapper{reader})
}
