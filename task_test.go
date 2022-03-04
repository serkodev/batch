package batch

import "testing"

func TestResult(t *testing.T) {
	t.Run("result get value", func(t *testing.T) {
		r := newTask[string, string]("foo")
		r.Return("foo", nil)
		v, err := r.(Result[string, string]).Result()
		assertEqual(t, v, "foo")
		assertEqual(t, err == nil, true)
	})

	t.Run("result get error", func(t *testing.T) {
		r := newTask[string, string]("foo").(Result[string, string])
		v, _ := r.Result()
		assertEqual(t, v, "")
		assertEqual(t, r.IsNoResult(), true)
	})
}
