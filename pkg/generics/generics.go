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

import "slices"

// Contains checks that the string is Contains in the specified list
func Contains[T comparable](s T, list []T) bool {
	return slices.Contains(list, s)
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
// The filter function is used to filter the items.
// Only the items that satisfy the picker function will be included in the result map.
func MakeMap[T any, K comparable, V any](items []T, namer func(T) K, mapper func(T, V) V, filter func(T) bool) map[K]V {
	result := make(map[K]V)

	Each(items, func(item T) {
		if filter != nil && !filter(item) {
			return
		}
		key := namer(item)
		current := result[key]
		result[key] = mapper(item, current)
	})
	return result
}

// Convert takes a slice of items of type T, a converter function that converts each item from type T to type V,
// and a filter function that determines whether each item should be included
func Convert[T any, V any](items []T, converter func(T) V, filter func(T) bool) []V {
	var result []V
	Each(items, func(item T) {
		if filter != nil && !filter(item) {
			return
		}
		result = append(result, converter(item))
	})
	return result
}

// Each applies the given action function to each item in the items slice.
// The action function takes one argument of type T and has no return value.
// Example usage:
//
//	Each([]int{1, 2, 3}, func(item int) {
//	    fmt.Println(item)
//	})
//
// Another example usage:
//
//	Each([]string{"apple", "banana", "cherry"}, func(item string) {
//	    fmt.Println(item)
//	})
func Each[T any](items []T, action func(T)) {
	for i := range items {
		item := items[i]
		action(item)
	}
}

// EachE applies the given action function to each item in the items slice.
// If the action function returns an error for any item, EachE immediately returns that error.
// Otherwise, it returns nil.
// It ensures that the action function has the signature func(T) error,
// where T represents the type of items in the slice.
func EachE[T any](items []T, action func(T) error) error {
	for i := range items {
		item := items[i]
		if err := action(item); err != nil {
			return err
		}
	}
	return nil
}
