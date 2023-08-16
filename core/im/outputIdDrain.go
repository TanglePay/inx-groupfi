package im

import "context"

type OutputIdDrainer struct {
	outputIdInput chan string
	consume       func(outputId string)
	fetchSize     int
}

func NewOutputIdDrainer(ctx context.Context, consume func(outputId string), concurrency int, chanSpace int, fetchSize int) *OutputIdDrainer {
	res := &OutputIdDrainer{
		outputIdInput: make(chan string, chanSpace),
		consume:       consume,
		fetchSize:     fetchSize,
	}
	for i := 0; i < concurrency; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					outputId := <-res.outputIdInput
					res.consume(outputId)
				}
			}
		}()
	}
	return res
}

// drain outputids, push to channel
func (drainer *OutputIdDrainer) Drain(outputIds []string) {
	for _, outputId := range outputIds {
		drainer.outputIdInput <- outputId
	}
}
