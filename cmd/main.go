package main

import (
	"aggregator/listener"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

// uni/usd
const ARB_UNI_PRICE_AGGREGATOR = "0xeFc5061B7a8AeF31F789F1bA5b3b8256674F2B71"

// eth/usd
const ARB_ETH_PRICE_AGGREGATOR = "0x3607e46698d218B3a5Cae44bF381475C0a5e2ca7"

// ETH/USD
const POLY_ETH_PRICE_AGGREGATOR = "0x4dD6655Ad5ed7C06c882f496E3f42acE5766cb89"

// UNI/USD
const POLY_UNI_PRICE_AGGREGATOR = "0x166816Cacb15f80badC5cd0cC24D64C8d1D1Cf61"

func main() {

	config.WithOptions(config.ParseEnv)

	// add driver for support yaml content
	config.AddDriver(yaml.Driver)

	err := config.LoadFiles("./config.yml")
	if err != nil {
		panic(err)
	}

	arbListener, err := listener.NewListenerByAPI(config.String("ARB_WEB3_API_KEY"))
	if err != nil {
		fmt.Println("Failed to open arb listener", err)
		return
	}
	fmt.Printf("Opened Listener. %s\n", arbListener.Description())

	arbListener.WatchPrice(listener.NewPriceAggregatorInfo(ARB_UNI_PRICE_AGGREGATOR, "UNI/USD", 8))
	arbListener.WatchPrice(listener.NewPriceAggregatorInfo(ARB_ETH_PRICE_AGGREGATOR, "ETH/USD", 8))

	polyListener, err := listener.NewListenerByAPI(config.String("POLY_WEB3_API_KEY"))
	if err != nil {
		fmt.Println("Failed to open poly listener", err)
		return
	}
	fmt.Printf("Opened Listener. %s\n", polyListener.Description())

	polyListener.WatchPrice(listener.NewPriceAggregatorInfo(POLY_UNI_PRICE_AGGREGATOR, "UNI/USD", 8))
	polyListener.WatchPrice(listener.NewPriceAggregatorInfo(POLY_ETH_PRICE_AGGREGATOR, "ETH/USD", 8))

	ftxListener, err := listener.NewFTXListener([]string{"ETH/USD", "UNI/USD"})
	if err != nil {
		fmt.Println("Failed to open ftx listener", err)
	}
	fmt.Printf("Opened Listener. %s\n", arbListener.Description())

	fmt.Println("start listen")
	aggregator := listener.NewListenerAggregator(
		[]listener.Listener{
			polyListener, arbListener,
			ftxListener,
		}, listener.NewPriceMap())

	aggregator.Listen()
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// go func() {
	<-c
	aggregator.Close()
	os.Exit(1)
	// }()

}
