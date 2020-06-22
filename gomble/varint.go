package gomble

// encode number to varint as specified in mumble protocol (Big-Endian!)
func encodeVarint(num int64) []byte {
	var maxVarint [10]byte // 10 Byte because of a negative recursive possible uin64 value
	offset := 0

	if (num < 0) {
		// it must be a negative number
		num = -num
		if (num <= 3) {
		// negative 2 Bit value,   111111xx
			maxVarint[0] = 0xFC | byte(num)
			return maxVarint[:1]
		} else {
		// negative value smaller than -3 
			maxVarint[0] = 0xF8
			offset = 1
		}
	}

	// 64 Bit Positive Number, 111101__ + long (64-bit)
	if (num > 4294967296) {
		maxVarint[0+offset] = 0xF4
		maxVarint[1+offset] = byte((num >> 56) & 0xFF)
		maxVarint[2+offset] = byte((num >> 48) & 0xFF)
		maxVarint[3+offset] = byte((num >> 40) & 0xFF)
		maxVarint[4+offset] = byte((num >> 32) & 0xFF)
		maxVarint[5+offset] = byte((num >> 24) & 0xFF)
		maxVarint[6+offset] = byte((num >> 16) & 0xFF)
		maxVarint[7+offset] = byte((num >> 8) & 0xFF)
		maxVarint[8+offset] = byte((num >> 0) & 0xFF)
		return maxVarint[:9]
	}
	// 32 Bit Positive Number, 111100__ + int (32-bit)
	if (num > 268435456) {
		maxVarint[0+offset] = 0xF0
		maxVarint[1+offset] = byte((num >> 24) & 0xFF)
		maxVarint[2+offset] = byte((num >> 16) & 0xFF)
		maxVarint[3+offset] = byte((num >> 8) & 0xFF)
		maxVarint[4+offset] = byte((num >> 0) & 0xFF)
		return maxVarint[:5]
	}
	// 28 Bit Positive Number, 1110xxxx + 3 bytes
	if (num > 2097151) {
		maxVarint[0+offset] = byte((num >> 24) | 0xE0)
		maxVarint[1+offset] = byte((num >> 16) & 0xFF)
		maxVarint[2+offset] = byte((num >> 8) & 0xFF)
		maxVarint[3+offset] = byte((num >> 0) & 0xFF)
		return maxVarint[:4]
	}
	// 21 Bit Positive Number, 110xxxxx + 2 bytes
	if (num > 16383) {
		maxVarint[0+offset] = byte((num >> 16) | 0xC0)
		maxVarint[1+offset] = byte((num >> 8) & 0xFF)
		maxVarint[2+offset] = byte((num >> 0) & 0xFF)
		return maxVarint[:3]
	}
	// 14 Bit Positive Number, 10xxxxxx + 1 byte
	if (num > 127) {
		maxVarint[0+offset] = byte((num >> 8) | 0x80)
		maxVarint[1+offset] = byte((num >> 0) & 0xFF)
		return maxVarint[:2]
	}
	// 7  Bit Positive Number, 0xxxxxxx 
	if (num > -1) {
		maxVarint[0+offset] = byte(num)
		return maxVarint[:1]
	}
	panic("you really shouldn't get here\n")
}

// Decodes a byte buffer into the respective int64 value according to mumble protocol specification of varint
// buffer len can be greater than the number it contains. The function just takes the first number it can decode out of buffer and ignores the rest content of buffer.
func decodeVarint(buffer []byte) (int64, error) {
	var num int64
	var offset int = 0
	var signed int64 = 1 // to save if it's a negative number. (-1 for negative value and 1 for positive)

	// byte inverted negative two bit number
	if (buffer[0] & 0xFC) == 0xFC {
		num = int64(buffer[0] & 0x03)
		return num, nil
	}
	// negative recursive varint
	if (buffer[0] & 0xF8) == 0xF8 {
		signed = -1
		offset++ // offset = 1 because first byte is indicator that it's a negative number
	}

	var nums []byte = buffer[offset:len(buffer)] // if it's a signed (negative) value, we take take only the slice of the number (not the first signed byte)

	// 64 bit number
	if (nums[0] & 0xF4) == 0xF4 {
		num = int64(nums[1]) << 56 | int64(nums[2]) << 48 | int64(nums[3]) << 40 | int64(nums[4]) << 32 | int64(nums[5]) << 24 | int64(nums[6]) << 16 | int64(nums[7]) << 8 | int64(nums[8])
		num *= signed
		return num, nil
	}
	// 32 bit number
	if (int64(nums[0]) & 0xF0) == 0xF0 {
		num = int64(nums[1]) << 24 | int64(nums[2]) << 16 | int64(nums[3]) << 8 | int64(nums[4])
		num *= signed
		return num, nil
	}
	// 28 bit number
	if (int64(nums[0]) & 0xE0) == 0xE0 {
		num = (int64(nums[0]) & 0x0F) << 24 | int64(nums[1]) << 16 | int64(nums[2]) << 8 | int64(nums[3])
		num *= signed
		return num, nil
	}
	// 21 bit number
	if (int64(nums[0]) & 0xC0) == 0xC0 {
		num = (int64(nums[0]) & 0x1F) << 16 | int64(nums[1]) << 8 | int64(nums[2])
		num *= signed
		return num, nil
	}
	// 14 bit number
	if (int64(nums[0]) & 0x80) == 0x80 {
		num = (int64(nums[0]) & 0x3F) << 8 | int64(nums[1])
		num *= signed
		return num, nil
	}
	// 7 bit positive number
	num = int64(nums[0]) & 0x7F
	num *= signed
	return num, nil
}
