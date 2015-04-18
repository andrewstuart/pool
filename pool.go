//Package pool implements synchronous access to a shared set of resources. If
//New is null or returns an error, then Pool will block until a resource
//becomes available.

package pool

import (
	"fmt"
	"time"
)

type Pooler interface {
	Get() (interface{}, error)
	Put(interface{})
}

var LimitReached error = fmt.Errorf("limit reached")
var Timeout = fmt.Errorf("timeout")

type NewFunc func() (interface{}, error)

//The Pool is the default/reference implementation of the pooler interface. It
//is safe for concurrent use. The New() function will never be called
//concurrently and so may access variables without worrying about concurrency
//protection
type Pool struct {
	New NewFunc

	is      []interface{}
	waiting []chan res

	newTimeout chan time.Duration
	newMax     chan uint
	get        chan (chan res)
	put        chan (interface{})
}

//NewPool takes a creator function and
func NewPool(nf NewFunc) *Pool {
	p := Pool{
		New:     nf,
		is:      make([]interface{}, 0, 1),
		waiting: make([]chan res, 0, 1),

		newTimeout: make(chan time.Duration),
		newMax:     make(chan uint),
		get:        make(chan (chan res)),
		put:        make(chan interface{}),
	}

	go p.run()

	return &p
}

type res struct {
	item interface{}
	err  error
}
type req struct {
	req chan res
	err error
}

//Get will return an interface from the pool, or attempt to create a new one if
//it cannot get one. If the New() function is nil or returns an error, it will
//wait for an interface{} to become available via Put(). These will be
//processed with FIFO semantics.
func (p *Pool) Get() (interface{}, error) {
	ch := make(chan res)
	p.get <- ch

	r := <-ch
	if r.err != nil {
		return nil, r.err
	} else {
		return r.item, nil
	}
}

func (p *Pool) Put(i interface{}) {
	p.put <- i
}

func (p *Pool) SetMax(max uint) {
	p.newMax <- max
}

func (p *Pool) SetTimeout(t time.Duration) {
	p.newTimeout <- t
}

func (p *Pool) run() {
	var max, created uint
	var timeout time.Duration
	remove := make(chan req)

	for {
		select {
		case max = <-p.newMax:
		case timeout = <-p.newTimeout:
		case toRemove := <-remove:
		search:
			for w := range p.waiting {
				if p.waiting[w] == toRemove.req {
					ch := p.waiting[w]
					p.waiting = append(p.waiting[:w], p.waiting[w:]...)

					if toRemove.err == nil {
						ch <- res{nil, Timeout}
					} else {
						ch <- res{nil, toRemove.err}
					}

					break search
				}
			}
		case getCh := <-p.get:
			var err error
			var v interface{}

			if len(p.is) > 0 {
				//if pool values are waiting
				last := len(p.is) - 1

				//Pop
				v = p.is[last]
				p.is = p.is[:last]

				getCh <- res{v, nil}
				close(getCh)
				continue
			} else if p.New != nil && (max == 0 || created < max) {
				//Try to get a new one
				v, err = p.New()
				if err == nil {
					created++

					//success
					getCh <- res{v, nil}
					close(getCh)
					continue
				}
			}

			//Error or New is nil
			p.waiting = append(p.waiting, getCh)

			if timeout > time.Duration(0) {
				go func() {
					time.Sleep(timeout)
					remove <- req{getCh, err}
				}()
			}
		case v := <-p.put:
			if len(p.waiting) > 0 {
				getCh := p.waiting[0]
				p.waiting = p.waiting[1:]

				getCh <- res{v, nil}
				close(getCh)
			} else {
				p.is = append(p.is, v)
			}
		}
	}
}
