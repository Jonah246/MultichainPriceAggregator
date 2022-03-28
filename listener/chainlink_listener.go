package listener

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

const BLOCK_CACHED_THRESHOLD = 100

type SubscriptionPool struct {
	subscriptions []ethereum.Subscription
}

func NewSubscriptionPool() SubscriptionPool {
	return SubscriptionPool{}
}

func (p *SubscriptionPool) Append(s ethereum.Subscription) {
	p.subscriptions = append(p.subscriptions, s)
}

func (p *SubscriptionPool) Close() {
	for _, s := range p.subscriptions {
		s.Unsubscribe()
	}
}

type ChainlinkListener struct {
	watch []PriceAggregatorInfo
	// TODO: consider refactor abi into aggregatorInfo
	abi abi.ABI

	pool    SubscriptionPool
	chainId int

	updatePriceChan chan listenerUpdatePrice

	blocks     []simpleBlock
	blockCount int

	client *ethclient.Client
	ctx    context.Context
}

type simpleBlock struct {
	blockNumber uint64
	blockTime   uint64
}

// func NewListener(client *ethclient.Client) *Listener {
// 	pool := NewSubscriptionPool()
// 	return &Listener{client: client, pool: pool}
// }

func NewListenerByAPI(rawurl string) (listener *ChainlinkListener, err error) {
	client, err := ethclient.Dial(rawurl)
	if err != nil {
		return nil, err
	}

	pool := NewSubscriptionPool()
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	abi, err := abi.JSON(strings.NewReader(OFFCHAIN_AGGREGATOR_ABI))
	if err != nil {
		return nil, err
	}

	return &ChainlinkListener{
		ctx:     context.Background(),
		client:  client,
		pool:    pool,
		abi:     abi,
		chainId: int(chainId.Int64()),
		blocks:  make([]simpleBlock, BLOCK_CACHED_THRESHOLD),
	}, nil
}

func (l *ChainlinkListener) WatchPrice(info PriceAggregatorInfo) {
	l.watch = append(l.watch, info)
}

func (l *ChainlinkListener) getWatchAddress() []common.Address {
	watchAddress := make([]common.Address, len(l.watch))
	for i, p := range l.watch {
		watchAddress[i] = p.contractAddress
	}
	return watchAddress
}

func (l *ChainlinkListener) getTimeByBlockNumber(blockNumber uint64) (uint64, error) {
	for i, b := range l.blocks {
		if i >= l.blockCount {
			break
		}
		if b.blockNumber == blockNumber {
			return b.blockTime, nil
		}
	}

	b, err := l.client.BlockByNumber(l.ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return 0, err
	}

	if l.blockCount >= BLOCK_CACHED_THRESHOLD {
		l.blockCount = 0
		l.blocks = make([]simpleBlock, BLOCK_CACHED_THRESHOLD)
	}
	l.blocks[l.blockCount] = simpleBlock{blockNumber: b.NumberU64(), blockTime: b.Time()}
	l.blockCount++

	fmt.Println("getBlock:", b.Time(), b.Number())

	return b.Time(), nil
}

func (l *ChainlinkListener) processLog(log types.Log) {
	for _, p := range l.watch {
		// do not watch too many contract in a single subscription
		if p.contractAddress == log.Address {
			if log.Topics[0] == common.HexToHash("0xf6a97944f31ea060dfde0566e4167c1a1082551e64b60ecb14d599a9d023d451") {
				// NewTransmission (index_topic_1 uint32 aggregatorRoundId, int192 answer, address transmitter, int192[] observations, bytes observers, bytes32 rawReportContext)View Source
				event, err := l.abi.EventByID(log.Topics[0])
				if err != nil {
					fmt.Print(err)
				}
				unpacked, err := event.Inputs.Unpack(log.Data)
				if err != nil {
					fmt.Println(err)
				}

				price, ok := unpacked[0].(*big.Int)
				if ok {
					decimalPrice := decimal.NewFromBigInt(price, int32(p.decimal)*-1)
					t, err := l.getTimeByBlockNumber(log.BlockNumber)
					if err != nil {
						fmt.Println(err)
					}
					l.updatePrice(p.description, decimalPrice, time.Unix(int64(t), 0))
				}

			}
		}
	}
}

func (l *ChainlinkListener) updatePrice(description string, price decimal.Decimal, updateTime time.Time) {
	// The aggregator should listen the updatePrice channel and handle it properly
	l.updatePriceChan <- listenerUpdatePrice{description: description, price: price, updateTime: updateTime}
}

func (l *ChainlinkListener) Description() string {
	return fmt.Sprintf("ChainId: %d", l.chainId)
}

func (l *ChainlinkListener) Listen() (chan listenerUpdatePrice, error) {
	// ListenTransmitEvent should only be called once.
	// This replace the updatePriceChan that the aggregator communicate with.
	logs := make(chan types.Log)
	query := ethereum.FilterQuery{
		Addresses: l.getWatchAddress(),
	}
	sub, err := l.client.SubscribeFilterLogs(l.ctx, query, logs)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			select {
			case <-sub.Err():
				return
			case vLog := <-logs:
				l.processLog(vLog) // pointer to event log
			}
		}
	}()
	l.pool.Append(sub)

	l.updatePriceChan = make(chan listenerUpdatePrice)
	return l.updatePriceChan, err
}

func (l *ChainlinkListener) Close() {
	l.Close()
}
