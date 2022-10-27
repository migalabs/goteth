package utils

// Check if bit n (0..7) is set where 0 is the LSB in little endian
func IsBitSet(input uint8, n int) bool {
	return (input & (1 << n)) > uint8(0)
}
