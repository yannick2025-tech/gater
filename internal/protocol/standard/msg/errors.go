// Package msg provides error definitions for message codec.
package msg

import "fmt"

// errInsufficientData 数据不足错误
func errInsufficientData(need, got int) error {
	return fmt.Errorf("insufficient data: need %d bytes, got %d", need, got)
}

// errFieldLength 字段长度错误
func errFieldLength(field string, expect, got int) error {
	return fmt.Errorf("field %s length mismatch: expect %d, got %d", field, expect, got)
}

// fmtHex 字节转十六进制字符串
func fmtHex(b []byte) string {
	return fmt.Sprintf("%X", b)
}

// StatusCodes 状态码（从standard包引用避免循环依赖，这里定义副本）
var StatusCodes = map[byte]string{
	0x00: "空闲-未插枪",
	0x0B: "空闲-已插枪",
	0x01: "充电中",
	0x02: "故障",
	0x04: "维护中",
	0x05: "离线",
	0x07: "升级中",
	0x08: "预约",
}
