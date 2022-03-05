package batch

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"
)

func assertEqual[T comparable](t *testing.T, a T, b T) {
	if a != b {
		t.Error("assert not equal:", a, b)
	}
}

func assertDuration(t *testing.T, duration time.Duration, want time.Duration) {
	tolerance := 10 * time.Millisecond
	if diff := duration - want; diff < 0 || diff > tolerance {
		t.Error("assert duration:", duration.Milliseconds(), "want:", want.Milliseconds())
	}
}

func TestSync(t *testing.T) {
	t.Parallel()

	db, fn := setupTestData(0, 2)
	a, _ := New(fn, 10*time.Nanosecond, 1).Run()

	// test sync query
	assertEqual(t, a.WaitTaskValue("key1"), "val1")
	assertEqual(t, a.WaitTaskValue("key2"), "val2")
	assertEqual(t, a.WaitTask("key3").IsNoResult(), true)

	// insert data
	db.Store("key3", "val3")
	assertEqual(t, a.WaitTaskValue("key3"), "val3")

	// GetChan
	{
		task := a.WaitTask("key1")
		task.Wait()
		assertEqual(t, task.ResultValue(), "val1")
	}

	// GetResult
	result, err := a.WaitTaskResult("key1")
	assertEqual(t, result, "val1")
	assertEqual(t, err == nil, true)

	// GetMulti
	results := a.WaitMultiTask([]string{"key1", "key2"})
	for i, result := range results { // avoid using reflect.DeepEqual with errors
		switch i {
		case 0:
			assertEqual(t, result.ResultValue(), "val1")
		case 1:
			assertEqual(t, result.ResultValue(), "val2")
		}
	}
}

func TestAsync(t *testing.T) {
	t.Parallel()

	max := 3
	_, fn := setupTestData(0, max)
	a, _ := New(func(ids TaskList[string, string]) {
		var keys []string
		for _, t := range ids {
			keys = append(keys, t.Value())
		}
		sort.Strings(keys)
		assertEqual(t, reflect.DeepEqual(keys, []string{"key1", "key2", "key3"}), true)

		fn(ids)
	}, 100*time.Millisecond, 100).Run()

	var w sync.WaitGroup
	w.Add(max)
	for i := 1; i <= max; i++ {
		go func(idx int) {
			defer w.Done()
			assertEqual(t, a.WaitTaskValue(fmt.Sprintf("key%d", idx)), fmt.Sprintf("val%d", idx))
		}(i)
	}
	w.Wait()
}

func TestWorker(t *testing.T) {
	processTime := 90 * time.Millisecond
	delayTime := 33 * time.Millisecond

	runTest := func(t *testing.T, a *Batch[string, string], wants map[string]time.Duration) {
		var w sync.WaitGroup
		w.Add(4)
		now := time.Now()
		for i := 1; i <= 4; i++ {
			go func(idx int) {
				defer w.Done()
				key := fmt.Sprintf("key%d", idx)
				assertEqual(t, a.WaitTaskValue(key), fmt.Sprintf("val%d", idx))
				assertDuration(t, time.Since(now), wants[key])
			}(i)
			time.Sleep(delayTime)
		}
		w.Wait()
	}

	_, fn := setupTestData(processTime, 4)

	assertEqual(t, canTestConcurrent(1), true)
	t.Run("1 worker", func(t *testing.T) {
		t.Parallel()

		a, _ := New(fn, 10*time.Second, 1).Run()
		runTest(t, a, map[string]time.Duration{
			"key1": processTime,
			"key2": processTime * 2,
			"key3": processTime * 3,
			"key4": processTime * 4,
		})
	})

	if canTestConcurrent(2) {
		t.Run("2 workers", func(t *testing.T) {
			t.Parallel()

			a, _ := New(fn, 10*time.Second, 1).RunWithWorkers(2)
			runTest(t, a, map[string]time.Duration{
				"key1": processTime,
				"key2": processTime + delayTime,
				"key3": (processTime * 2),
				"key4": (processTime * 2) + delayTime,
			})
		})
	} else {
		t.Log("not enough procs to test 2 workers")
	}

	if canTestConcurrent(4) {
		t.Run("4 workers", func(t *testing.T) {
			t.Parallel()

			a, _ := New(fn, 10*time.Second, 1).RunWithWorkers(4)
			runTest(t, a, map[string]time.Duration{
				"key1": processTime,
				"key2": processTime + delayTime,
				"key3": processTime + delayTime*2,
				"key4": processTime + delayTime*3,
			})
		})
	} else {
		t.Log("not enough procs to test 4 workers")
	}
}

func TestUnlimitWait(t *testing.T) {
	t.Run("wait forever, never reach max query", func(t *testing.T) {
		t.Parallel()
		max := 2
		_, fn := setupTestData(0, max)
		a, _ := New(fn, NeverFlushTimeout, 3).Run()
		var w sync.WaitGroup
		w.Add(max)
		for i := 1; i <= max; i++ {
			go func(idx int) {
				defer w.Done()
				select {
				case <-a.Task(fmt.Sprintf("key%d", idx)).Done():
					t.Error("expect not return value")
				case <-time.After(100 * time.Millisecond):
					break
				}
			}(i)
		}
		w.Wait()
	})

	t.Run("wait forever, reach max query", func(t *testing.T) {
		t.Parallel()
		max := 2
		_, fn := setupTestData(0, max)
		a, _ := New(fn, NeverFlushTimeout, 2).Run()
		var w sync.WaitGroup
		w.Add(max)
		for i := 1; i <= max; i++ {
			go func(idx int) {
				defer w.Done()
				task := a.Task(fmt.Sprintf("key%d", idx))
				select {
				case <-task.Done():
					assertEqual(t, task.ResultValue(), fmt.Sprintf("val%d", idx))
					break
				case <-time.After(100 * time.Millisecond):
					t.Error("expect flush by max query")
				}
			}(i)
		}
		w.Wait()
	})
}

func TestUnlimitMax(t *testing.T) {
	t.Parallel()

	max := 10
	flushTimeout := 500 * time.Millisecond
	_, fn := setupTestData(0, max)
	a, _ := New(fn, flushTimeout, 0).Run()

	var w sync.WaitGroup
	w.Add(max)
	now := time.Now()
	for i := 1; i <= max; i++ {
		go func(idx int) {
			defer w.Done()
			assertEqual(t, a.WaitTaskValue(fmt.Sprintf("key%d", idx)), fmt.Sprintf("val%d", idx))
			assertDuration(t, time.Since(now), flushTimeout)
		}(i)
	}
	w.Wait()
}

func TestAggregate(t *testing.T) {
	t.Parallel()

	a, _ := New(func(ids TaskList[string, string]) {
		assertEqual(t, len(ids), 9)

		m := ids.Group()
		m["key1"].Return("val1", nil)
		m["key2"].Return("val2", nil)
		m["key3"].Return("val3", nil)
	}, 100*time.Millisecond, 100).Run()

	max := 3
	var w sync.WaitGroup
	for i := 1; i <= max; i++ {
		for j := 0; j < max; j++ {
			w.Add(1)
			go func(idx int) {
				defer w.Done()
				assertEqual(t, a.WaitTaskValue(fmt.Sprintf("key%d", idx)), fmt.Sprintf("val%d", idx))
			}(i)
		}
	}
	w.Wait()
}

func canTestConcurrent(concurrent int) bool {
	return runtime.GOMAXPROCS(0) >= concurrent && runtime.NumCPU() >= concurrent
}

func setupTestData(fakeDuration time.Duration, dbItemsCount int) (sync.Map, func(ids TaskList[string, string])) {
	// generate sample db
	var db sync.Map
	for i := 1; i <= dbItemsCount; i++ {
		db.Store(fmt.Sprintf("key%d", i), fmt.Sprintf("val%d", i))
	}

	// generate fn
	fn := func(ids TaskList[string, string]) {
		if fakeDuration > 0 {
			time.Sleep(fakeDuration)
		}
		for _, id := range ids {
			if r, ok := db.Load(id.Value()); ok {
				id.Return(r.(string), nil)
			}
		}
	}
	return db, fn
}
