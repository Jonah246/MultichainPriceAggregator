package listener

// It's heavily reference to https://github.com/go-numb/go-ftx
import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

const FTX_URL = "wss://ftx.com/ws/"

const (
	UNDEFINED = iota
	ERROR
	TICKER
	TRADES
	ORDERBOOK
	ORDERS
	FILLS
)

type FTXListener struct {
	ctx             context.Context
	updatePriceChan chan listenerUpdatePrice

	symbols []string
	conn    *websocket.Conn
}

func subscribe(conn *websocket.Conn, channels, symbols []string) error {
	if symbols != nil {
		for i := range channels {
			for j := range symbols {
				if err := conn.WriteJSON(&request{
					Op:      "subscribe",
					Channel: channels[i],
					Market:  symbols[j],
				}); err != nil {
					return err
				}
			}
		}
	} else {
		for i := range channels {
			if err := conn.WriteJSON(&request{
				Op:      "subscribe",
				Channel: channels[i],
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

type request struct {
	Op      string `json:"op"`
	Channel string `json:"channel"`
	Market  string `json:"market"`
}

type FtxTime struct {
	Time time.Time
}

func (p *FtxTime) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}

	sec, nsec := math.Modf(f)
	p.Time = time.Unix(int64(sec), int64(nsec))
	return nil
}

type Ticker struct {
	Bid     float64 `json:"bid"`
	Ask     float64 `json:"ask"`
	BidSize float64 `json:"bidSize"`
	AskSize float64 `json:"askSize"`
	Last    float64 `json:"last"`
	Time    FtxTime `json:"time"`
}

type Response struct {
	Type   int
	Symbol string

	Ticker Ticker

	Results error
}

func unsubscribe(conn *websocket.Conn, channels, symbols []string) error {
	if symbols != nil {
		for i := range channels {
			for j := range symbols {
				if err := conn.WriteJSON(&request{
					Op:      "unsubscribe",
					Channel: channels[i],
					Market:  symbols[j],
				}); err != nil {
					return err
				}
			}
		}
	} else {
		for i := range channels {
			if err := conn.WriteJSON(&request{
				Op:      "unsubscribe",
				Channel: channels[i],
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func ping(conn *websocket.Conn) (err error) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, []byte(`{"op": "pong"}`)); err != nil {
				return err
			}
		}
	}
}

func (l *FTXListener) processTick(res Response) {
	l.updatePriceChan <- listenerUpdatePrice{
		description: res.Symbol,
		price:       decimal.NewFromFloat32(float32(res.Ticker.Last)),
		updateTime:  res.Ticker.Time.Time,
	}

}

func (l *FTXListener) Listen() (chan listenerUpdatePrice, error) {
	conn, _, err := websocket.DefaultDialer.Dial(FTX_URL, nil)

	channels := []string{"ticker"}
	symbols := []string{"ETH/USD", "UNI/USD"}
	if err := subscribe(conn, channels, symbols); err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	go ping(conn)

	go func() {
		defer conn.Close()
		defer unsubscribe(conn, channels, symbols)
		for {
			var res Response
			_, msg, err := conn.ReadMessage()

			typeMsg, err := jsonparser.GetString(msg, "type")
			if typeMsg == "error" {
				fmt.Printf("[ERROR]: error: %+v", string(msg))
				res.Type = ERROR
				res.Results = fmt.Errorf("%v", string(msg))
				continue
			}
			if err != nil {
				fmt.Printf("[ERROR]: data err: %v %s", err, string(msg))

			}

			channel, err := jsonparser.GetString(msg, "channel")
			if err != nil {
				fmt.Printf("[ERROR]: channel error: %+v", string(msg))

			}
			if channel != "ticker" {
				continue
			}
			market, err := jsonparser.GetString(msg, "market")
			if err != nil {
				fmt.Printf("[ERROR]: market err: %+v", string(msg))
				res.Type = ERROR
				res.Results = fmt.Errorf("%v", string(msg))
			}

			res.Symbol = market

			data, _, _, err := jsonparser.Get(msg, "data")
			if err != nil {
				if isSubscribe, _ := jsonparser.GetString(msg, "type"); isSubscribe == "subscribed" {
					fmt.Printf("[SUCCESS]: %s %+v", isSubscribe, string(msg))
					continue
				}
			}
			if err := json.Unmarshal(data, &res.Ticker); err != nil {
				fmt.Printf("[WARN]: cant unmarshal ticker %+v", err)
			}
			l.processTick(res)
		}
	}()
	l.conn = conn
	l.updatePriceChan = make(chan listenerUpdatePrice)
	return l.updatePriceChan, nil
}

func (l *FTXListener) Close() {
	l.conn.Close()
}

func (l *FTXListener) Description() string {
	return "FTX"
}

func NewFTXListener(symbols []string) (*FTXListener, error) {

	return &FTXListener{symbols: symbols}, nil
}
