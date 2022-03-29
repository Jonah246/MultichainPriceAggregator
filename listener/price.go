package listener

import (
	"time"

	"github.com/buger/goterm"
	"github.com/shopspring/decimal"
)

type PriceSource struct {
	// Source: [Arbitrum/ Polygon/ Coinbase]
	Source string
	// Description: [ETH/USD, UNI/USD]
	Description string
	// TODO: fixed point? bignumber
	Price decimal.Decimal

	UpdateTime time.Time
}

type sourcePair struct {
	Source      string
	Description string
}

type PriceMap struct {
	priceList []PriceSource

	// logging
	priceUpdateCount map[sourcePair]int
}

func NewPriceMap() *PriceMap {
	return &PriceMap{priceUpdateCount: make(map[sourcePair]int)}
}

func (m *PriceMap) UpdatePrice(description string, price decimal.Decimal, source string, updateTime time.Time) {
	for i, p := range m.priceList {
		if description == p.Description && source == p.Source {
			p.Price = price
			p.UpdateTime = updateTime
			m.priceList[i] = p
			m.priceUpdateCount[sourcePair{Source: source, Description: description}]++
			// only Update one item at a time
			return
		}
	}

	p := PriceSource{
		Source:      source,
		Price:       price,
		Description: description,
		UpdateTime:  updateTime,
	}

	m.priceUpdateCount[sourcePair{Source: source, Description: description}] = 1

	m.priceList = append(m.priceList, p)
}

func (m *PriceMap) PrintPrice() {
	goterm.Clear()

	goterm.MoveCursor(1, 2)
	goterm.Printf("-----------------------------------------------------------\n")
	goterm.Printf("_._._._._._._._._._._._._._._._._._._._._._._._._._._._._._.\n")

	for _, p := range m.priceList {
		fprice, _ := p.Price.Float64()
		goterm.Printf("Source: %s Description: %s, Price: %f UpdateCount: %d UpdateTime: %s\n",
			p.Source, p.Description, fprice, m.priceUpdateCount[sourcePair{Source: p.Source, Description: p.Description}], p.UpdateTime.String())
	}
	goterm.Printf("_._._._._._._._._._._._._._._._._._._._._._._._._._._._._._.\n")
	goterm.Printf("-----------------------------------------------------------\n\n")
	goterm.Flush()
}
