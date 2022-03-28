package listener

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type ListenerAggregator struct {
	mu     sync.RWMutex
	prices *PriceMap

	listeners []Listener
}

func NewListenerAggregator(listeners []Listener, prices *PriceMap) *ListenerAggregator {
	return &ListenerAggregator{
		mu:        sync.RWMutex{},
		prices:    prices,
		listeners: listeners,
	}
}

func (a *ListenerAggregator) NewUpdatePrice(description string, price decimal.Decimal, source string, updateTime time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.prices.UpdatePrice(description, price, source, updateTime)
}

func (a *ListenerAggregator) PrintPrice() {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.prices.PrintPrice()
}

func (a *ListenerAggregator) Close() {
	for _, l := range a.listeners {
		l.Close()
	}
}

func (a *ListenerAggregator) Listen() error {
	for _, l := range a.listeners {
		c, err := l.Listen()
		if err != nil {
			// This is not safe to do
			// Should gracefully close the connection here.
			return err
		}
		// TODO: close the channel gracefully
		go func(c chan listenerUpdatePrice, sourceDescription string) {
			for {
				select {
				case updatePrice := <-c:
					a.NewUpdatePrice(updatePrice.description, updatePrice.price, sourceDescription, updatePrice.updateTime)
					a.PrintPrice()
				default:
				}
			}
		}(c, l.Description())
	}
	return nil
}
