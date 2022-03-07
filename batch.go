package batch

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Batch is the instance, it contains the flush and worker settings.
// Create an instance of Batch, by using New()
type Batch[T comparable, R any] struct {
	// Max waiting time of worker to flush tasks.
	MaxWait time.Duration

	// Max amount of tasks, the worker will flush after reaching it.
	MaxSize int

	fn     func(TaskList[T, R])
	ch     chan TaskInput[T, R]
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
func New[T comparable, R any](fn func(TaskList[T, R]), flushMaxWait time.Duration, flushMaxSize int) *Batch[T, R] {
	a := &Batch[T, R]{
		MaxWait: flushMaxWait,
		MaxSize: flushMaxSize,
		fn:      fn,
		ch:      make(chan TaskInput[T, R]),
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
	flushChan := make(chan []TaskInput[T, R], workers)
	for i := 0; i < workers; i++ {
		go a.runWorker(flushChan)
	}

	// flush timer
	timer := time.NewTimer(a.MaxWait)
	for {
		// wait task to start
		task := <-a.ch

		a.debugf("[task] start")
		tasks := []TaskInput[T, R]{task}

		// stop timer
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}

		if a.MaxSize != 1 {
			if a.MaxWait != NeverFlushTimeout {
				// reset timer to count down
				timer.Reset(a.MaxWait)

			wait:
				for {
					select {
					case task := <-a.ch:
						a.debugf("[task] append")
						tasks = append(tasks, task)
						if a.MaxSize > 0 && len(tasks) >= a.MaxSize {
							a.debugf("[flush] max task reached")
							break wait
						}
					case <-timer.C:
						a.debugf("[flush] timeout")
						break wait
					}
				}
			} else {
				// loop without timer
				for task := range a.ch {
					a.debugf("[task] append")
					tasks = append(tasks, task)
					if a.MaxSize > 0 && len(tasks) >= a.MaxSize {
						a.debugf("[flush] max task reached")
						break
					}
				}
			}
		}

		// flush
		flushChan <- tasks
	}
}

func (a *Batch[T, R]) runWorker(flushChan <-chan []TaskInput[T, R]) {
	for tasks := range flushChan {
		// execute
		a.fn(tasks)

		// return results
		// TODO
		// for _, task := range tasks {
		// 	task.finish()
		// }
	}
}

// Get with a key and return with a Result[R] channel.
func (a *Batch[T, R]) Task(item T) TaskOutput[R] {
	t := newTask[T, R](item)
	a.ch <- t
	return t.(TaskOutput[R])
}

// Get with a key and return with a Result[R] synchronously.
// It is a shortcut for <-GetChan(key)
func (a *Batch[T, R]) WaitTask(item T) Result[R] {
	t := a.Task(item)
	return t.Wait()
}

// Get with a key and return with Value and Error of Result[R] synchronously.
// It is a shortcut for Get(key).Get()
func (a *Batch[T, R]) WaitTaskResult(item T) (R, error) {
	return a.WaitTask(item).Get()
}

// Get with a key and return with Value Result[R] synchronously.
// It is a shortcut for Get(key).Value
func (a *Batch[T, R]) WaitTaskValue(item T) R {
	return a.WaitTask(item).GetValue()
}

// Get with multiple keys and return a slice of Result[R] synchronously.
func (a *Batch[T, R]) WaitMultiTask(items []T) []Result[R] {
	output := make([]Result[R], len(items))
	var w sync.WaitGroup
	w.Add(len(items))
	for i, key := range items {
		go func(i int, item T) {
			defer w.Done()
			output[i] = a.WaitTask(item)
		}(i, key)
	}
	w.Wait()
	return output
}
