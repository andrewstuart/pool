//Package pool implements synchronous access to a shared set of resources. If
//New is null or returns an error, then Pool will block until a resource
//becomes available.

package pool

import "fmt"

type Pooler interface {
	Get() interface{}
	Put(interface{})
}

var LimitReached error = fmt.Errorf("limit reached")

type NewFunc func() (interface{}, error)

//The Pool is the default/reference implementation of the pooler interface. It
//is safe for concurrent use. The New() function will never be called
//concurrently and so may access variables without worrying about concurrency
//protection
type Pool struct {
	New          NewFunc
	max, created uint

	is      []interface{}
	waiting []chan interface{}

	newMax chan uint
	get    chan (chan interface{})
	put    chan (interface{})
}

//NewPool takes a creator function and
func NewPool(nf NewFunc) *Pool {
	p := Pool{
		New:     nf,
		is:      make([]interface{}, 0, 1),
		waiting: make([]chan interface{}, 0, 1),

		newMax: make(chan uint),
		get:    make(chan (chan interface{})),
		put:    make(chan interface{}),
	}

	go p.run()

	return &p
}

//Get will return an interface from the pool, or attempt to create a new one if
//it cannot get one. If the New() function is nil or returns an error, it will
//wait for an interface{} to become available via Put(). These will be
//processed with FIFO semantics.
func (p *Pool) Get() interface{} {
	ch := make(chan interface{})
	p.get <- ch
	return <-ch
}

func (p *Pool) Put(i interface{}) {
	p.put <- i
}

func (p *Pool) SetMax(max uint) {
	p.newMax <- max
}

func (p *Pool) run() {
	for {
		select {
		case newMax := <-p.newMax:
			p.max = newMax
		case getCh := <-p.get:
			if len(p.is) > 0 {
				//if pool values are waiting
				last := len(p.is) - 1

				//Pop
				v := p.is[last]
				p.is = p.is[:last]

				getCh <- v
				close(getCh)
			} else if p.New != nil && (p.max == 0 || p.created < p.max) {
				//Try to get a new one
				if v, err := p.New(); err == nil {
					p.created++

					//success
					getCh <- v
					close(getCh)
				} else {
					//error = wait
					p.waiting = append(p.waiting, getCh)
				}
			} else {
				p.waiting = append(p.waiting, getCh)
			}
		case v := <-p.put:
			if len(p.waiting) > 0 {
				getCh := p.waiting[0]
				p.waiting = p.waiting[1:]

				getCh <- v
				close(getCh)
			} else {
				p.is = append(p.is, v)
			}
		}
	}
}
