package ginradix

func longestCommonPrefix(a, b string) string {
	l := min(len(a), len(b))
	for i := 0; i < l; i++ {
		if a[i] != b[i] {
			return a[0:i]
		}
	}
	return a[0:l]
}
