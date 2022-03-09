package batch

import "testing"

func BenchmarkBatch(b *testing.B) {
	a, _ := New(func(ids TaskList[int, string]) {
		ids.Return("ok", nil)
	}, NeverFlushTimeout, b.N).Run()
	for i := 0; i < b.N-1; i++ {
		a.Task(i)
	}
	a.Task(b.N).Wait()
}
