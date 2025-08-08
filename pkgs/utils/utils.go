package utils

func IfElse[T any](test bool, yes, no T) T {
	if test {
		return yes
	}
	return no
}

func DefaultIfZero[T comparable](x, d T) T {
	var zero T
	if x == zero {
		return d
	}
	return x
}
