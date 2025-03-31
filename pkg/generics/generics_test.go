package generics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		item     string
		list     []string
		expected bool
	}{
		{
			name:     "found in list",
			item:     "apple",
			list:     []string{"apple", "banana", "cherry"},
			expected: true,
		},
		{
			name:     "not found in list",
			item:     "orange",
			list:     []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "empty list",
			item:     "apple",
			list:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.item, tt.list)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMakItemMap(t *testing.T) {
	type testItem struct {
		id   string
		name string
	}

	items := []testItem{
		{id: "1", name: "item1"},
		{id: "2", name: "item2"},
	}

	result := MakItemMap(items, func(i testItem) string { return i.id })

	assert.Equal(t, 2, len(result))
	assert.Equal(t, "item1", result["1"].name)
	assert.Equal(t, "item2", result["2"].name)
}

func TestMakeMap(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	// Test with filter
	result := MakeMap(items,
		func(i int) string { return "key" + string(rune('0'+i)) },
		func(i int, v int) int { return i * 2 },
		func(i int) bool { return i%2 == 0 },
	)

	assert.Equal(t, 2, len(result))
	assert.Equal(t, 4, result["key2"])
	assert.Equal(t, 8, result["key4"])
}

func TestConvert(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	// Test with filter
	result := Convert(items,
		func(i int) string { return string(rune('A' + i - 1)) },
		func(i int) bool { return i%2 == 0 },
	)

	assert.Equal(t, 2, len(result))
	assert.Equal(t, "B", result[0])
	assert.Equal(t, "D", result[1])
}

func TestEach(t *testing.T) {
	items := []int{1, 2, 3}
	sum := 0

	Each(items, func(i int) {
		sum += i
	})

	assert.Equal(t, 6, sum)
}

func TestEachE(t *testing.T) {
	items := []int{1, 2, 3}

	t.Run("success case", func(t *testing.T) {
		sum := 0
		err := EachE(items, func(i int) error {
			sum += i
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 6, sum)
	})

	t.Run("error case", func(t *testing.T) {
		expectedErr := errors.New("test error")
		sum := 0
		err := EachE(items, func(i int) error {
			if i == 2 {
				return expectedErr
			}
			sum += i
			return nil
		})

		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, sum) // Should stop at 2
	})
}
