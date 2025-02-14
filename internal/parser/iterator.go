package parser

// iterator wraps a slice and provides iteration methods
type iterator[T any] struct {
	items []T
	index int
}

// newIterator creates a new iterator
func newIterator[T any](items []T) *iterator[T] {
	return &iterator[T]{items: items, index: -1}
}

func (it *iterator[T]) Peek() (T, bool) {
	var zeroValue T

	if it.index+1 < 0 || it.index+1 >= len(it.items) {
		return zeroValue, false
	}

	return it.items[it.index+1], true
}

// Next advances to the next item and returns it
func (it *iterator[T]) Next() (T, bool) {
	var zeroValue T

	if it.index+1 >= len(it.items) {
		return zeroValue, false
	}

	it.index++ // Move to next item
	return it.items[it.index], true
}

// HasNext checks if there are more items left
func (it *iterator[T]) HasNext() bool {
	return it.index+1 < len(it.items)
}

// Current returns the current item without advancing
func (it *iterator[T]) Current() (T, bool) {
	var zeroValue T

	if it.index < 0 || it.index >= len(it.items) {
		return zeroValue, false
	}
	return it.items[it.index], true
}

// Reset resets the iterator to the beginning
func (it *iterator[T]) Reset() {
	it.index = -1
}
