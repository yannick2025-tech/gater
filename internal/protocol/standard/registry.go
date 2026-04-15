// Package standard provides the message registry implementation.
package standard

import (
	"fmt"

	"github.com/yannick2025-tech/nts-gater/internal/protocol/types"
)

type registryKey struct {
	FuncCode  byte
	Direction types.Direction
}

type messageRegistryImpl struct {
	factories map[registryKey]func() types.Message
	specs     map[registryKey]types.MessageSpec
}

func newRegistry() *messageRegistryImpl {
	return &messageRegistryImpl{
		factories: make(map[registryKey]func() types.Message),
		specs:     make(map[registryKey]types.MessageSpec),
	}
}

func (r *messageRegistryImpl) Register(funcCode byte, dir types.Direction, factory func() types.Message) {
	key := registryKey{FuncCode: funcCode, Direction: dir}
	msg := factory()
	r.factories[key] = factory
	r.specs[key] = msg.Spec()
}

func (r *messageRegistryImpl) Create(funcCode byte, dir types.Direction) (types.Message, bool) {
	key := registryKey{FuncCode: funcCode, Direction: dir}
	f, ok := r.factories[key]
	if !ok {
		return nil, false
	}
	return f(), true
}

func (r *messageRegistryImpl) Spec(funcCode byte, dir types.Direction) (types.MessageSpec, bool) {
	key := registryKey{FuncCode: funcCode, Direction: dir}
	s, ok := r.specs[key]
	return s, ok
}

func (r *messageRegistryImpl) AllSpecs() []types.MessageSpec {
	result := make([]types.MessageSpec, 0, len(r.specs))
	for _, s := range r.specs {
		result = append(result, s)
	}
	return result
}

func (r *messageRegistryImpl) NeedReply(funcCode byte, dir types.Direction) bool {
	s, ok := r.Spec(funcCode, dir)
	if !ok {
		return false
	}
	return s.NeedReply
}

func (r *messageRegistryImpl) ReplyDirection(dir types.Direction) types.Direction {
	switch dir {
	case types.DirectionUpload:
		return types.DirectionReply
	case types.DirectionDownload:
		return types.DirectionReply
	default:
		panic(fmt.Sprintf("cannot determine reply direction for %v", dir))
	}
}
