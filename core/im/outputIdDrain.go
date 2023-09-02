package im

import "context"

type ItemDrainer struct {
	itemInput chan interface{}
	consume   func(item interface{})
	fetchSize int
}

func NewItemDrainer(ctx context.Context, consume func(item interface{}), concurrency int, chanSpace int, fetchSize int) *ItemDrainer {
	res := &ItemDrainer{
		itemInput: make(chan interface{}, chanSpace),
		consume:   consume,
		fetchSize: fetchSize,
	}
	for i := 0; i < concurrency; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					item := <-res.itemInput
					res.consume(item)
				}
			}
		}()
	}
	return res
}

// Drain items and push to channel
func (drainer *ItemDrainer) Drain(items []interface{}) {
	for _, item := range items {
		drainer.itemInput <- item
	}
}
