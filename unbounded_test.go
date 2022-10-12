package channels

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestUnboundedChannelReaderBlock(t *testing.T) {
	defer goleak.VerifyNone(t)

	ch := NewUnbounded[int]()
	defer closeChannel(t, ch)

	go func() {
		time.Sleep(50 * time.Millisecond)
		ch.In() <- 1
	}()

	// Reading from an empty channel should block.
	got := <-ch.Out()
	assert.Equal(t, 1, got)
}

func TestUnboundedChannelWriteThenRead(t *testing.T) {
	defer goleak.VerifyNone(t)

	const elementsCount = 100
	ch := NewUnbounded[int]()
	defer closeChannel(t, ch)

	// Send elements to the channel.
	go func() {
		for i := 0; i < elementsCount; i++ {
			ch.In() <- i
		}
	}()

	require.Eventually(t, func() bool {
		return ch.Len() == elementsCount
	}, 50*time.Millisecond, 5*time.Millisecond)

	// Read all elements from the channel.
	for i := 0; i < elementsCount; i++ {
		gotEl := <-ch.Out()
		assert.Equal(t, i, gotEl)
		assert.Equal(t, elementsCount-i-1, ch.Len())
	}
}

func TestUnboundedChannelReadWriteConcurrently(t *testing.T) {
	defer goleak.VerifyNone(t)

	const elementsCount = 100
	ch := NewUnbounded[int]()
	defer closeChannel(t, ch)

	writeFinished := make(chan struct{})
	readFinished := make(chan struct{})

	// Send elements to the channel.
	go func() {
		for i := 0; i < elementsCount; i++ {
			ch.In() <- i
		}
		close(writeFinished)
	}()

	// Read all elements from the channel.
	go func() {
		for i := 0; i < elementsCount; i++ {
			<-ch.Out()
		}
		close(readFinished)
	}()

	for writeFinished != nil || readFinished != nil {
		select {
		case <-writeFinished:
			writeFinished = nil
		case <-readFinished:
			readFinished = nil
		}
	}

	require.Equal(t, 0, ch.Len())
}

func closeChannel[T any](t *testing.T, ch *Unbounded[T]) {
	t.Helper()

	ch.Close()
	select {
	case _, ok := <-ch.closed:
		if !ok {
			return
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("timeout: channel should be closed")
	}
}
