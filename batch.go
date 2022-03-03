package batch

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type Task[T, R any] struct {
	value  T
	result R
	err    error
	ch     chan Result[R]
}

func (t *Task[T, R]) Value() T {
	return t.value
}

func (t *Task[T, R]) Return(r R) {
	t.result = r
}

func (t *Task[T, R]) Error(err error) {
	t.err = err
}

// Batch is the instance, it contains the flush and worker settings.
// Create an instance of Batch, by using New()
type Batch[T, R any] struct {
	// Max waiting time of worker to flush query tasks.
	MaxWait time.Duration

	// Max amount of query tasks, the worker will flush after reaching it.
	MaxSize int

	fn     func([]Task[T, R]) error
	ch     chan Task[T, R]
	once   sync.Once
	debugf func(format string, v ...interface{})
}

// If NeverFlushTimeout is set for the flushMaxWait, the aggregator
// will never flush with timeout.
var NeverFlushTimeout time.Duration = -1

// New creates a new Aggregator. The flushMaxWait variable
// sets the maximum timeout of flushing. If flushMaxWait equals to
// NeverFlushTimeout then the aggregator will never flush with timeout.
// The flushMaxSize variable sets the maximum size of task.
// If the flushMaxSize <= 0, the aggregator will never flush with amount of tasks.
func New[T, R any](fn func([]Task[T, R]) error, flushMaxWait time.Duration, flushMaxSize int) *Batch[T, R] {
	a := &Batch[T, R]{
		MaxWait: flushMaxWait,
		MaxSize: flushMaxSize,
		fn:      fn,
		ch:      make(chan Task[T, R]),
		debugf:  func(format string, v ...interface{}) {},
	}
	return a
}

// If Debug() is called, the Aggregator will prints debug messages.
func (a *Batch[T, R]) Debug() *Batch[T, R] {
	a.debugf = log.New(os.Stdout, fmt.Sprintf("[batch]"), 0).Printf
	return a
}

// Start run Aggregator with single worker.
func (a *Batch[T, R]) Run() (*Batch[T, R], error) {
	return a.RunWithWorkers(1)
}

// Start run Aggregator with n workers.
func (a *Batch[T, R]) RunWithWorkers(workers int) (*Batch[T, R], error) {
	if a.fn == nil {
		return a, errors.New("nil aggregator fn")
	}
	a.once.Do(func() {
		go a.run(workers)
	})
	return a, nil
}

func (a *Batch[T, R]) run(workers int) {
	// workers
	if workers < 1 {
		workers = 1
	}
	flushChan := make(chan []Task[T, R], workers)
	for i := 0; i < workers; i++ {
		go a.runWorker(flushChan)
	}

	// flush timer
	timer := time.NewTimer(a.MaxWait)
	for {
		// wait task to start
		task := <-a.ch

		a.debugf("[query] start")
		tasks := []Task[T, R]{task}

		// stop timer
		if !timer.Stop() && len(timer.C) > 0 {
			<-timer.C
		}

		if a.MaxSize != 1 {
			if a.MaxWait != NeverFlushTimeout {
				// reset timer to count down
				timer.Reset(a.MaxWait)
			}
		wait:
			for {
				select {
				case task := <-a.ch:
					a.debugf("[query] insert")
					tasks = append(tasks, task)
					if a.MaxSize > 0 && len(tasks) >= a.MaxSize {
						a.debugf("[flush] max query reached")
						break wait
					}
				case <-timer.C:
					a.debugf("[flush] timeout")
					break wait
				}
			}
		}

		// flush
		flushChan <- tasks
	}
}

func (a *Batch[T, R]) runWorker(flushChan <-chan []Task[T, R]) {
	for tasks := range flushChan {
		// execute
		err := a.fn(tasks)

		// return results
		for _, task := range tasks {
			var r Result[R]
			r.Error = task.err
			r.Value = task.result
			if err != nil {
				r.Error = err
			}
			task.ch <- r
			close(task.ch)
		}
	}
}

// Query with a key and return with a Result[R] channel.
func (a *Batch[T, R]) QueryChan(item T) <-chan Result[R] {
	ch := make(chan Result[R], 1)
	a.ch <- Task[T, R]{
		value: item,
		ch:    ch,
	}
	return ch
}

// Query with a key and return with a Result[R] synchronously.
// It is a shortcut for <-QueryChan(key)
func (a *Batch[T, R]) Query(item T) Result[R] {
	return <-a.QueryChan(item)
}

// Query with a key and return with Value and Error of Result[R] synchronously.
// It is a shortcut for Query(key).Get()
func (a *Batch[T, R]) QueryResult(item T) (R, error) {
	return a.Query(item).Get()
}

// Query with a key and return with Value Result[R] synchronously.
// It is a shortcut for Query(key).Value
func (a *Batch[T, R]) QueryValue(item T) R {
	return a.Query(item).Value
}

// Query with multiple keys and return a slice of Result[R] synchronously.
func (a *Batch[T, R]) QueryMulti(items []T) []Result[R] {
	output := make([]Result[R], len(items))
	var w sync.WaitGroup
	w.Add(len(items))
	for i, key := range items {
		go func(i int, item T) {
			defer w.Done()
			output[i] = a.Query(item)
		}(i, key)
	}
	w.Wait()
	return output
}
