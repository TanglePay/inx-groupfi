package im

import (
	"context"
	"sync"
)

type ItemDrainer struct {
	itemInput chan interface{}
	consume   func(item interface{})
	FetchSize int
	ctx       context.Context
	wg        *sync.WaitGroup
}

func NewItemDrainer(ctx context.Context, consume func(item interface{}), concurrency int, chanSpace int, fetchSize int) *ItemDrainer {
	res := &ItemDrainer{
		itemInput: make(chan interface{}, chanSpace),
		consume:   consume,
		FetchSize: fetchSize,
		ctx:       ctx,
		wg:        &sync.WaitGroup{},
	}
	for i := 0; i < concurrency; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case item := <-res.itemInput:
					res.consume(item)
					res.wg.Done()
				}
			}
		}()
	}
	return res
}

// Drain items and push to channel
func (drainer *ItemDrainer) Drain(items []interface{}) {
	drainer.wg.Add(len(items))
	for _, item := range items {
		select {
		case <-drainer.ctx.Done():
			return
		case drainer.itemInput <- item:
		}
	}
}

// Wait for all items to be processed or context to be done
func (drainer *ItemDrainer) Wait() {
	done := make(chan struct{})
	go func() {
		drainer.wg.Wait()
		close(done)
	}()
	select {
	case <-drainer.ctx.Done():
		return
	case <-done:
		return
	}
}
