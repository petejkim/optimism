package rollup

import (
	"errors"
	"math"
)

type GetTargetGasPerSecond func() float64

type GasPricer struct {
	curPrice              float64
	floorPrice            float64
	getTargetGasPerSecond GetTargetGasPerSecond
	maxChangePerEpoch     float64
}

// LinearInterpolation can be used to dynamically update target gas per second
func GetLinearInterpolationFn(getX func() float64, x1 float64, x2 float64, y1 float64, y2 float64) func() float64 {
	return func() float64 {
		return y1 + ((getX()-x1)/(x2-x1))*(y2-y1)
	}
}

func NewGasPricer(curPrice float64, floorPrice float64, getTargetGasPerSecond GetTargetGasPerSecond, maxPercentChangePerEpoch float64) (*GasPricer, error) {
	if floorPrice < 1 {
		return nil, errors.New("floorPrice must be greater than or equal to 1")
	}
	if maxPercentChangePerEpoch <= 0 || maxPercentChangePerEpoch > 100 {
		return nil, errors.New("maxPercentChangePerEpoch must be between (0,100].")
	}
	return &GasPricer{
		curPrice:              math.Max(curPrice, floorPrice),
		floorPrice:            floorPrice,
		getTargetGasPerSecond: getTargetGasPerSecond,
		maxChangePerEpoch:     maxPercentChangePerEpoch / 100,
	}, nil
}

// Calculate the next gas price given some average gas per second over the last epoch
func (p *GasPricer) CalcNextEpochGasPrice(avgGasPerSecondLastEpoch float64) (float64, error) {
	targetGasPerSecond := p.getTargetGasPerSecond()
	if avgGasPerSecondLastEpoch < 0 {
		return 0.0, errors.New("avgGasPerSecondLastEpoch cannot be negative.")
	}
	if targetGasPerSecond < 1 {
		return 0.0, errors.New("gasPerSecond cannot be less than 1.")
	}
	// The percent difference between our current average gas & our target gas
	proportionOfTarget := avgGasPerSecondLastEpoch / targetGasPerSecond
	// The percent that we should adjust the gas price to reach our target gas
	proportionToChangeBy := 0.0
	if proportionOfTarget >= 1 { // If average avgGasPerSecondLastEpoch is GREATER than our target
		proportionToChangeBy = math.Min(proportionOfTarget, 1+p.maxChangePerEpoch)
	} else {
		proportionToChangeBy = math.Max(proportionOfTarget, 1-p.maxChangePerEpoch)
	}
	return math.Ceil(math.Max(p.floorPrice, p.curPrice*proportionToChangeBy)), nil
}

// End the current epoch and update the current gas price for the next epoch.
func (p *GasPricer) CompleteEpoch(avgGasPerSecondLastEpoch float64) (float64, error) {
	gp, err := p.CalcNextEpochGasPrice(avgGasPerSecondLastEpoch)
	if err != nil {
		return gp, err
	}
	p.curPrice = gp
	return gp, nil
}