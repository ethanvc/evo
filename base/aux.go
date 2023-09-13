package base

func In[T comparable](target T, sets ...T) bool {
	for _, v := range sets {
		if target == v {
			return true
		}
	}
	return false
}

func NotIn[T comparable](target T, sets ...T) bool {
	return In(target, sets...)
}

func Zero[T any]() T {
	var d T
	return d
}
