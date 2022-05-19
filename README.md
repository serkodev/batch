# batch

Go batch process library. Comparing to [aggregator](https://github.com/serkodev/aggregator/), it is more flexible and able to process in batch with different pattern.

> Since this library is currently used internally, the public document is not completed, more contents will be added soon.

### THIS PROJECT IS IN BETA

This project may contain bugs and have not being tested at all. Use under your own risk, but feel free to test, make pull request and improve this project.

## Install

Currently Go 1.18+ is required (for go generics)

```
go get github.com/serkodev/batch@latest
```

## Samples

### Batch List

```go
New(func(ids TaskList[string, string]) {
    ids.Return("val1", nil)
}, 100*time.Millisecond, 100).Run()
```

### Batch Map

```go
New(func(ids TaskList[string, string]) {
    m := ids.Group()
    m["key1"].Return("val1", nil)
    m["key2"].Return("val2", nil)
    m["key3"].Return("val3", nil)
}, 100*time.Millisecond, 100).Run()
```

### Fallback

```go
a2, _ := New(func(ids TaskList[string, string]) {
    m := ids.Group()
    m["key1"].Return("val1", nil)
    m["key2"].Return("val2", nil)
    m["key3"].Return("val3", nil)
}, 50*time.Millisecond, 10).Run()

a1, _ := New(func(ids TaskList[string, string]) {
    m := ids.Group()
    m["key1"].Return("val1", nil)
    m["key2"].Return("val2", nil)
    m["key3"].Fallback(a2)
}, 50*time.Millisecond, 10).Run()
```

## Usage

See [batch_test.go](./batch_test.go).

## Performance

See [BENCHMARKS.md](./BENCHMARKS.md).

## LICENSE

MIT
