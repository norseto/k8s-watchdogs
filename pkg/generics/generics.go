/*
MIT License

Copyright (c) 2024 Norihiro Seto

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

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

// MakItemMap takes in a slice of items and a namer function,
// and returns a map that maps the result of the namer function
// for each item to the item itself.
func MakItemMap[T any, K comparable](items []T, namer func(T) K) map[K]T {
	return MakeMap(items, namer, func(i, _ T) T { return i }, nil)
}

// MakeMap creates a map by iterating through the items and applying the namer, mapper, and picker functions.
// It returns a map with keys of type K and values of type V.
// The namer function is used to determine the key for each item.
// The mapper function is used to compute the value for each key. The current value for the same key is passed
// as the second argument to the mapper function.
// The picker function is used to filter the items.
// Only the items that satisfy the picker function will be included in the result map.
func MakeMap[T any, K comparable, V any](items []T, namer func(T) K, mapper func(T, V) V, picker func(T) bool) map[K]V {
	result := make(map[K]V)
	for _, i := range items {
		if picker != nil && !picker(i) {
			continue
		}
		key := namer(i)
		current := result[key]
		result[key] = mapper(i, current)
	}
	return result
}
