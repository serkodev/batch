package batch

type TaskInput[T comparable, R any] interface {
	Value() T
	Return(R, error)
	Fallback(*Batch[T, R])
	returned() bool
}

type TaskOutput[R any] interface {
	Wait() Result[R]
	Done() <-chan Result[R]
}

type task[T comparable, R any] struct {
	value T
	ch    chan Result[R]
	rt    bool
}

func newTask[T comparable, R any](value T, ch chan Result[R]) TaskInput[T, R] {
	return &task[T, R]{
		value: value,
		ch:    ch,
	}
}

func (t *task[T, R]) Value() T {
	return t.value
}

func (t *task[T, R]) Return(r R, err error) {
	if !t.rt {
		t.rt = true
		t.ch <- Result[R]{r, err}
		close(t.ch)
	}
}

func (t *task[T, R]) Fallback(batch *Batch[T, R]) {
	if !t.rt {
		t.rt = true
		batch.taskWithChan(t.value, t.ch)
	}
}

func (t *task[T, R]) returned() bool {
	return t.rt
}

func (t *task[T, R]) Done() <-chan Result[R] {
	return t.ch
}

func (t *task[T, R]) Wait() Result[R] {
	return <-t.Done()
}
