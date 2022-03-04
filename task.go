package batch

type Task[T, R any] interface {
	Value() T
	Return(R, error)

	finish()
}

type Result[T, R any] interface {
	Wait()
	Done() <-chan struct{}

	Result() (R, error)
	ResultValue() R
	IsNoResult() bool
}

type task[T, R any] struct {
	value T
	ch    chan struct{}

	// result
	returned bool
	result   R
	err      error
}

func newTask[T any, R any](value T) Task[T, R] {
	return &task[T, R]{
		value: value,
		ch:    make(chan struct{}, 1),
	}
}

func (t *task[T, R]) Value() T {
	return t.value
}

func (t *task[T, R]) Return(r R, err error) {
	t.returned = true
	t.result = r
	t.err = err
}

func (t *task[T, R]) finish() {
	t.ch <- struct{}{}
	close(t.ch)
}

func (t *task[T, R]) IsNoResult() bool {
	return !t.returned
}

func (t *task[T, R]) Result() (R, error) {
	return t.result, t.err
}

func (t *task[T, R]) ResultValue() R {
	r, _ := t.Result()
	return r
}

func (t *task[T, R]) Done() <-chan struct{} {
	return t.ch
}

func (t *task[T, R]) Wait() {
	<-t.Done()
}
