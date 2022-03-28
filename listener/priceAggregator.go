package listener

import "github.com/ethereum/go-ethereum/common"

type PriceAggregatorInfo struct {
	contractAddress common.Address
	decimal         int
	description     string
	chainId         int
}

func NewPriceAggregatorInfo(hexAddress string, description string, decimal int) PriceAggregatorInfo {
	return PriceAggregatorInfo{
		contractAddress: common.HexToAddress(hexAddress),
		description:     description,
		decimal:         decimal,
	}
}
