package pipeline

func (p *PipelineNode) Pipe(children ...*PipelineNode) *PipelineNode {
	p.Children = children
	return p
}
