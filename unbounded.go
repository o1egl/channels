package channels

import "container/list"

type Unbounded[T any] struct {
	input, output chan T
	length        chan int
	buffer        *list.List
	// closed is used to signal that the channel is closed. Used for testing.
	closed chan struct{}
}

func NewUnbounded[T any]() *Unbounded[T] {
	ch := &Unbounded[T]{
		input:  make(chan T),
		output: make(chan T),
		length: make(chan int),
		closed: make(chan struct{}),
		buffer: list.New(),
	}
	go ch.infiniteBuffer()
	return ch
}

func (ch *Unbounded[T]) In() chan<- T {
	return ch.input
}

func (ch *Unbounded[T]) Out() <-chan T {
	return ch.output
}

func (ch *Unbounded[T]) Len() int {
	return <-ch.length
}

func (ch *Unbounded[T]) Close() {
	close(ch.input)
}

func (ch *Unbounded[T]) infiniteBuffer() {
	var input, output chan T
	var (
		nextElement *list.Element
		nextValue   T
	)

	input = ch.input

	for input != nil || output != nil {
		select {
		case elem, open := <-input:
			if open {
				ch.buffer.PushBack(elem)
			} else {
				input = nil
			}
		case output <- nextValue:
			ch.buffer.Remove(nextElement)
		case ch.length <- ch.buffer.Len():
		}

		if ch.buffer.Len() > 0 {
			output = ch.output
			nextElement = ch.buffer.Front()
			nextValue = nextElement.Value.(T)
		} else {
			output = nil
			nextElement = nil
		}
	}

	close(ch.output)
	close(ch.length)
	close(ch.closed)
}
