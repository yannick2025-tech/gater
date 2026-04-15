// Package codec provides checksum calculation for protocol frames.
package codec

// CalcChecksum 计算校验和
// 协议规定：采用累积和校验，取低字节。校验范围为数据域（加密后的数据域）。
// 如果校验和的数值超过0xFF，则取其补码作为校验和。
func CalcChecksum(data []byte) byte {
	if len(data) == 0 {
		return 0
	}

	sum := 0
	for _, b := range data {
		sum += int(b)
	}

	// 取低字节
	checksum := sum & 0xFF

	// 如果超过0xFF，取补码
	if sum > 0xFF {
		checksum = (^checksum + 1) & 0xFF
	}

	return byte(checksum)
}

// VerifyChecksum 校验校验和
func VerifyChecksum(data []byte, expected byte) bool {
	return CalcChecksum(data) == expected
}
