package utils

// Map maps a list of type P to a list of type Q, using the provided mapper function
func Map[P, Q any](list []P, mapper func(P) Q) []Q {
	result := make([]Q, len(list))

	for i, p := range list {
		result[i] = mapper(p)
	}

	return result
}

func MapEntries[P comparable, Q, R any](m map[P]Q, mapper func(P, Q) R) []R {
	result := make([]R, len(m))

	i := 0
	for k, v := range m {
		result[i] = mapper(k, v)
	}

	return result
}
