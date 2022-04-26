package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRanges(t *testing.T) {
	testRanges := []string{"0:100", "A:100", "100:B"}

	r1, err := NewRangeFromString(testRanges[0])
	require.Equal(t, nil, err)
	require.Equal(t, 0, r1.min)
	require.Equal(t, 100, r1.max)

	num1 := r1.GetRandomNumber()
	if (num1 > r1.max) || (num1 < r1.min) {
		t.Errorf("random value %d not in range %s", num1, testRanges[0])
	}

	_, err = NewRangeFromString(testRanges[1])
	require.NotEqual(t, nil, err)

	_, err = NewRangeFromString(testRanges[2])
	require.NotEqual(t, nil, err)

}
