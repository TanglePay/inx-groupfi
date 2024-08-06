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
	cancel    context.CancelFunc
	wg        *sync.WaitGroup
	closeOnce sync.Once
}

func NewItemDrainer(ctx context.Context, consume func(item interface{}), concurrency int, chanSpace int, fetchSize int) *ItemDrainer {
	// Create a derived context from the global context
	ctx, cancel := context.WithCancel(ctx)
	res := &ItemDrainer{
		itemInput: make(chan interface{}, chanSpace),
		consume:   consume,
		FetchSize: fetchSize,
		ctx:       ctx,
		cancel:    cancel,
		wg:        &sync.WaitGroup{},
	}
	for i := 0; i < concurrency; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-res.itemInput:
					if !ok {
						return
					}
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
	for idx, item := range items {
		select {
		case <-drainer.ctx.Done():
			// Ensure wg.Done() is called for each remaining item
			for i := idx; i < len(items); i++ {
				drainer.wg.Done()
			}
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

// Close stops all goroutines and releases the channel
func (drainer *ItemDrainer) Close() {
	drainer.closeOnce.Do(func() {
		drainer.cancel()
		close(drainer.itemInput)
	})
}
