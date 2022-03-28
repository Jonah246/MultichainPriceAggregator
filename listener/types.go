package listener

import (
	"time"

	"github.com/shopspring/decimal"
)

type listenerUpdatePrice struct {
	description string
	price       decimal.Decimal
	updateTime  time.Time
}
