package testdata

func f(xs []int) {
	for x := range xs {
		_ = x
	}
}
