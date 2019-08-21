package redisutil

type Result struct {
	Lines    uint64
	BadCount uint64
	Errors   map[string]uint64
}

func (r Result) AddError(emes string) {
	r.BadCount++
	r.Errors[emes]++
}

func (r Result) Combine(o Result) Result {
	ret := Result{
		Lines:    r.Lines + o.Lines,
		BadCount: r.BadCount + o.BadCount,
		Errors:   r.Errors,
	}

	for k, _ := range o.Errors {
		r.Errors[k] += o.Errors[k]
	}

	return ret
}

func NewResult() Result {
	return Result{
		Errors: map[string]uint64{},
	}
}
