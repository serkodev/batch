package batch

import (
	"sync"
)

type Task[T, R any] interface {
	Get() T
	Set(R, error)
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
	result     R
	err        error
	isSet      bool
	returnOnce sync.Once
}

func newTask[T any, R any](value T) Task[T, R] {
	return &task[T, R]{
		value: value,
		ch:    make(chan struct{}, 1),
	}
}

func (t *task[T, R]) Get() T {
	return t.value
}

func (t *task[T, R]) Set(r R, err error) {
	t.isSet = true
	t.result = r
	t.err = err
}

func (t *task[T, R]) Return(r R, err error) {
	t.Set(r, err)
	t.finish()
}

func (t *task[T, R]) finish() {
	t.returnOnce.Do(func() {
		t.ch <- struct{}{}
		close(t.ch)
	})
}

func (t *task[T, R]) IsNoResult() bool {
	return !t.isSet
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
