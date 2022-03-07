package batch

type Result[R any] struct {
	Value R
	Error error
}

func (r Result[R]) Get() (R, error) {
	return r.Value, r.Error
}

func (r Result[R]) GetValue() R {
	v, _ := r.Get()
	return v
}
