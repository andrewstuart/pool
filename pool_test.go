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

	c, err := p.Get()

	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}

	if c.(int) != 1 {
		t.Errorf("Value!=1")
	}

	p.Put(c)

	c, err = p.Get()
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}

	if c.(int) != 1 {
		t.Errorf("Wrong value: %d, should be 1", c)
	}

	c, _ = p.Get()

	if c.(int) != 2 {
		t.Errorf("Wrong value of c. %d, should be 2", c)
	}

	delay := 200 * time.Millisecond
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		start := time.Now()
		c, _ = p.Get()
		delta := time.Now().Sub(start)

		if delta < delay {
			t.Error("Got a value before the delay of %v", time.Second)
		}

		if c.(int) != 2 {
			t.Errorf("Wrong value of c: %d, should be 2", c)
		}
		wg.Done()
	}()

	time.Sleep(delay)
	p.Put(c)

	wg.Wait()

	p.SetTimeout(time.Millisecond)

	now := time.Now()

	q, err := p.Get()

	if time.Now().Sub(now) < time.Millisecond {
		t.Errorf("Exited earlier than the timeout")
	}

	if q != nil {
		t.Errorf("Should not have gotten anything.")
	}

	if err != Timeout {
		t.Errorf("Did not send the \"Timeout\" error. %v", err)
	}
}
