package rollup

import (
	"errors"
)

type GetLatestBlockNumberFn func() (uint64, error)
type UpdateL2GasPriceFn func(float64) error

type GasPriceUpdater struct {
	gasPricer              GasPricer
	epochStartBlockNumber  uint64
	averageBlockGasLimit   uint64
	epochLengthSeconds     uint64
	getLatestBlockNumberFn GetLatestBlockNumberFn
	updateL2GasPriceFn     UpdateL2GasPriceFn
}

func GetAverageGasPerSecond(
	epochStartBlockNumber uint64,
	latestBlockNumber uint64,
	epochLengthSeconds uint64,
	averageBlockGasLimit uint64,
) float64 {
	return float64((latestBlockNumber - epochStartBlockNumber) * averageBlockGasLimit / epochLengthSeconds)
}

func NewGasPriceUpdater(
	gasPricer *GasPricer,
	epochStartBlockNumber uint64,
	averageBlockGasLimit uint64,
	epochLengthSeconds uint64,
	getLatestBlockNumberFn GetLatestBlockNumberFn,
	updateL2GasPriceFn UpdateL2GasPriceFn,
) (*GasPriceUpdater, error) {
	if epochStartBlockNumber < 0 {
		return nil, errors.New("epochStartBlockNumber must be non-negative.")
	}
	if averageBlockGasLimit < 1 {
		return nil, errors.New("averageBlockGasLimit cannot be less than 1 gas.")
	}
	if epochLengthSeconds < 1 {
		return nil, errors.New("epochLengthSeconds cannot be less than 1 second.")
	}
	return &GasPriceUpdater{
		gasPricer:              *gasPricer,
		epochStartBlockNumber:  epochStartBlockNumber,
		epochLengthSeconds:     epochLengthSeconds,
		averageBlockGasLimit:   averageBlockGasLimit,
		getLatestBlockNumberFn: getLatestBlockNumberFn,
		updateL2GasPriceFn:     updateL2GasPriceFn,
	}, nil
}

func (g *GasPriceUpdater) UpdateGasPrice() error {
	latestBlockNumber, err := g.getLatestBlockNumberFn()
	if err != nil {
		return err
	}
	if latestBlockNumber < g.epochStartBlockNumber {
		return errors.New("Latest block number less than the last epoch's block number.")
	}
	averageGasPerSecond := GetAverageGasPerSecond(
		g.epochStartBlockNumber,
		latestBlockNumber,
		g.epochLengthSeconds,
		g.averageBlockGasLimit,
	)
	_, err = g.gasPricer.CompleteEpoch(averageGasPerSecond)
	if err != nil {
		return err
	}
	g.epochStartBlockNumber = latestBlockNumber
	err = g.updateL2GasPriceFn(g.gasPricer.curPrice)
	if err != nil {
		return err
	}
	return nil
}
