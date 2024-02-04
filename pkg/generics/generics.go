package generics

// Contains checks that the string is Contains in the specified list
func Contains[T comparable](s T, list []T) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// MakeMap takes in a slice of items and a namer function,
// and returns a map that maps the result of the namer function
// for each item to the item itself.
func MakeMap[T any, K comparable](items []T, namer func(T) K) map[K]T {
	result := make(map[K]T)
	for _, i := range items {
		result[namer(i)] = i
	}
	return result
}
