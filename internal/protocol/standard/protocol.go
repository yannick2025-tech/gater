package standard

import (
	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

// StandardProtocol 标准充电桩通信协议实现
type StandardProtocol struct {
	frameConfig  types.FrameConfig
	cryptoConfig types.CryptoConfig
	tempConfig   types.TempConfig
	registry     *messageRegistryImpl
	fixedKeyCodes []byte
}

// New 创建标准协议实例
func New() *StandardProtocol {
	p := &StandardProtocol{
		frameConfig: types.NewDefaultFrameConfig(),
		cryptoConfig: types.CryptoConfig{
			Algorithm:          "AES256-CBC-PKCS7",
			FixedKey:           "4845727543536D5570716A3843596451",
			IVRule:             "last_16_bytes_of_key",
			ZeroLengthNoEncrypt: true,
		},
		tempConfig: types.TempConfig{
			ValidMin:      0,
			ValidMax:      250,
			Offset:        -40,
			AbnormalValue: 0xFE,
			InvalidValue:  0xFF,
		},
		fixedKeyCodes: []byte{0x0A, 0x0B, 0x21},
		registry:      newRegistry(),
	}
	p.registerMessages()
	return p
}

func (p *StandardProtocol) Name() string          { return "standard" }
func (p *StandardProtocol) Version() byte         { return 0x06 }
func (p *StandardProtocol) FrameConfig() types.FrameConfig   { return p.frameConfig }
func (p *StandardProtocol) CryptoConfig() types.CryptoConfig { return p.cryptoConfig }
func (p *StandardProtocol) TempConfig() types.TempConfig     { return p.tempConfig }
func (p *StandardProtocol) Registry() types.MessageRegistry  { return p.registry }

func (p *StandardProtocol) IsFixedKeyFuncCode(code byte) bool {
	for _, c := range p.fixedKeyCodes {
		if c == code {
			return true
		}
	}
	return false
}
