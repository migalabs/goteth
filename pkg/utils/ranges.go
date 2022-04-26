package utils

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Range struct {
	min int
	max int
}

func NewRange(min int, max int) *Range {
	return &Range{
		min: min,
		max: max,
	}
}

func NewRangeFromString(strRange string) (*Range, error) {
	// parse the range string
	ranges := strings.Split(strRange, ":")
	if len(ranges) < 2 {
		return nil, errors.New(fmt.Sprintf("unable to parse range no MIN:MAX format - %s", strRange))
	}

	// get int from string
	min, err := strconv.Atoi(ranges[0])
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to parse MIN value, non numerical - %s", ranges[0]))
	}
	max, err := strconv.Atoi(ranges[1])
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to parse MAX value, non numerical - %s", ranges[1]))
	}

	return &Range{
		min: min,
		max: max,
	}, nil
}

// Methods

func (r *Range) GetRandomNumber() int {
	return rand.Intn(r.max-r.min) + r.min
}

func (r *Range) GetRandomNumberStr() string {
	return fmt.Sprintf("%d", rand.Intn(r.max-r.min)+r.min)
}

// Rage utils
func IsValidRangeuint64(init uint64, final uint64) bool {
	if init < 0 || final < 0 {
		return false
	}
	return final > init
}
