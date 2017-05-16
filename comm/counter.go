//author tyf
//date   2017-02-10 14:20
//desc 

package comm

import (
	"sync"
)

type Counter struct {
	mu    *sync.Mutex
	count int32
}

func NewCounter() *Counter {
	return &Counter{
		mu: &sync.Mutex{},
		count: 0,
	}
}

func (p *Counter) AddOne() {
	p.mu.Lock()
	p.count++
	//atomic.AddInt32(&p.count, 1)
	p.mu.Unlock()
}

func (p *Counter) Count() int32 {
	return p.count
}

func (p *Counter) Reset() {
	p.mu.Lock()
	p.count = 0
	p.mu.Unlock()
}
