package pool

import (
	"sync"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	i := 0

	nf := NewFunc(func() (interface{}, error) {
		if i < 2 {
			i++
			return i, nil
		} else {
			return nil, LimitReached
		}
	})

	p := NewPool(nf)

	c := p.Get().(int)

	if c != 1 {
		t.Errorf("Value!=1")
	}

	p.Put(c)
	c = p.Get().(int)

	if c != 1 {
		t.Errorf("Wrong value: %d, should be 1", c)
	}

	c = p.Get().(int)

	if c != 2 {
		t.Errorf("Wrong value of c. %d, should be 2", c)
	}

	delay := 200 * time.Millisecond
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		start := time.Now()
		c = p.Get().(int)
		delta := time.Now().Sub(start)

		if delta < delay {
			t.Error("Got a value before the delay of %v", time.Second)
		}

		if c != 2 {
			t.Errorf("Wrong value of c: %d, should be 2", c)
		}
		wg.Done()
	}()

	time.Sleep(delay)
	p.Put(c)

	wg.Wait()
}
