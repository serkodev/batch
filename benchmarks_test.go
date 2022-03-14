package batch

import (
	"log"
	"strconv"
	"sync"
	"testing"
	"time"
)

func BenchmarkBatch(b *testing.B) {
	a, _ := New(func(ids TaskList[int, string]) {
		ids.Return("ok", nil)
	}, NeverFlushTimeout, b.N).Run()
	for i := 0; i < b.N-1; i++ {
		a.Task(i)
	}
	a.Task(b.N).Wait()
}

// see: https://github.com/graph-gophers/dataloader/blob/master/dataloader_test.go

var avg = &Avg{}

func BenchmarkBatchDataLoader(b *testing.B) {
	a, _ := New(func(ids TaskList[string, interface{}]) {
		avg.Add(len(ids))
		ids.Return(nil, nil)
	}, 16*time.Millisecond, 0).Run()
	for i := 0; i < b.N; i++ {
		a.Task(strconv.Itoa(i))
	}
	log.Printf("avg: %f", avg.Avg())
}

type Avg struct {
	total  float64
	length float64
	lock   sync.RWMutex
}

func (a *Avg) Add(v int) {
	a.lock.Lock()
	a.total += float64(v)
	a.length++
	a.lock.Unlock()
}

func (a *Avg) Avg() float64 {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if a.total == 0 {
		return 0
	} else if a.length == 0 {
		return 0
	}
	return a.total / a.length
}
