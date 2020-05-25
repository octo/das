package dkb4q

func encodeReport(reportType byte, data []byte) []byte {
	encLen := 2 + len(data) + 1

	// The output buffer size must be divisible by 7.
	bufSize := encLen
	if rem := bufSize % 7; rem != 0 {
		bufSize += 7 - rem
	}

	enc := make([]byte, bufSize, bufSize)
	enc[0] = reportType
	enc[1] = byte(encLen - 2)
	copy(enc[2:encLen-1], data)
	enc[encLen-1] = xorAll(enc[:encLen-1])

	return enc
}

func xorAll(data []byte) byte {
	var ret byte
	for _, d := range data {
		ret ^= d
	}
	return ret
}

func isZero(data []byte) bool {
	for _, b := range data {
		if b != 0x00 {
			return false
		}
	}
	return true
}
