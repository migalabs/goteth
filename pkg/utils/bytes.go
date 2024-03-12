package utils

func CountConsecutiveEnding0(s []byte) int {

	phraseLen := len(s)
	count := 0

	for i := phraseLen - 1; i >= 0; i-- {
		singleByte := s[i]
		if singleByte == 0 {
			count += 1
		} else {
			break
		}
	}

	return count
}
