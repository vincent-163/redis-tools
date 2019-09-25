// rtools provide useful utilities for redis.
package rtools

import (
	"context"
	"sync"

	"github.com/gomodule/redigo/redis"
)

// Pipeline wraps a redis.Conn to provide pipelining. It can be used by concurrent callers, so it may be a good idea to use it instead of a connection pool for non-blocking commands.
type Pipeline struct {
	conn redis.Conn
	mu sync.Mutex
	ch chan interface{}
	closeOnce sync.Once
}

// Callback function to receive results.
type Callback func(interface{}, error)

type flusher chan error

// Create a new pipeline.
func NewPipeline(conn redis.Conn) *Pipeline {
	p := &Pipeline{
		conn: conn,
		ch: make(chan interface{}, 5000),
	}
	go p.run()
	return p
}

func (p *Pipeline) run() {
	var curErr error
	for obj := range p.ch {
		switch val := obj.(type) {
		case Callback:
			data, err := p.conn.Receive()
			if err != nil {
				if _, ok := err.(redis.Error); !ok {
					curErr = err
				}
			}
			val(data, err)
		case flusher:
			val <- curErr
		}
	}
}

// Perform a command. Errors are deferred until a call to Flush(). Note that while callbacks will be invoked in order, they may be called in a separate goroutine than the one calling `Do()`.
func (p *Pipeline) Do(callback Callback, cmdName string, args ...interface{}) {
	p.mu.Lock()
	p.conn.Send(cmdName, args...)
	p.ch <- callback
	p.mu.Unlock()
}

// Flushes all previous commands and waits for them to finish. If the error is nil, all previous commands will be performed and the callback will be invoked with either the result or a Redis error.
func (p *Pipeline) Flush(ctx context.Context) error {
	if err := p.conn.Flush(); err != nil {
		return err
	}
	f := flusher(make(chan error, 1))
	p.ch <- f
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-f:
		return err
	}
}

// Closes the connection and releases all resources. Previously registered callbacks will eventually be called.
func (p *Pipeline) Close() error {
	p.closeOnce.Do(func() {
		p.conn.Close()
		close(p.ch)
	})
	return p.conn.Err()
}
