package utils

// AsciiCompareFold compares two slices. if lengths do not match, the difference will be discarded.
func AsciiCompareFold(a, b []byte) int {
	minLen := len(b)
	if len(a) < minLen {
		minLen = len(a)
	}
	for i := 0; i < minLen; i++ {
		aChar := a[i]
		bChar := b[i]
		if aChar > 0x40 && aChar < 0x5B {
			aChar = aChar | 0x20
		}
		if bChar > 0x40 && bChar < 0x5B {
			bChar = bChar | 0x20
		}
		if aChar != bChar {
			if aChar < bChar {
				return -1
			}
			if aChar > bChar {
				return 1
			}
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}
