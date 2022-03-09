package batch

type TaskInput[T, R any] interface {
	Value() T
	Return(R, error)
}

type TaskOutput[R any] interface {
	Wait() Result[R]
	Done() <-chan Result[R]
}

type task[T, R any] struct {
	value T
	ch    chan Result[R]
}

func newTask[T any, R any](value T) TaskInput[T, R] {
	return &task[T, R]{
		value: value,
		ch:    make(chan Result[R], 1),
	}
}

func (t *task[T, R]) Get() T {
	return t.value
}

func (t *task[T, R]) Return(r R, err error) {
	if t.ch != nil {
		t.ch <- Result[R]{r, err}
		close(t.ch) // TODO: need close?
	}
}

func (t *task[T, R]) Done() <-chan Result[R] {
	return t.ch
}

func (t *task[T, R]) Wait() Result[R] {
	return <-t.Done()
}

// TODO: use sync.pool reuse task
