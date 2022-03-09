package batch

type TaskList[T comparable, R any] []TaskInput[T, R]

func (list TaskList[T, R]) Values() []T {
	vals := make([]T, len(list))
	for i, t := range list {
		vals[i] = t.Value()
	}
	return vals
}

func (list TaskList[T, R]) Return(r R, err error) {
	for _, t := range list {
		t.Return(r, err)
	}
}

func (list TaskList[T, R]) Group() TaskMap[T, R] {
	m := make(TaskMap[T, R])
	for _, t := range list {
		k := t.Value()
		m[k] = append(m[k], t)
	}
	return m
}

type TaskMap[T comparable, R any] map[T]TaskList[T, R]

func (m TaskMap[T, R]) Keys() []T {
	keys := make([]T, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func (m TaskMap[T, R]) Return(r R, err error) {
	for _, list := range m {
		list.Return(r, err)
	}
}
