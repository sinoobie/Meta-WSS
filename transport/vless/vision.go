package vless

import (
	"bytes"
	"encoding/binary"
	"math/rand"

	"github.com/Dreamacro/clash/common/buf"
	"github.com/Dreamacro/clash/log"

	"github.com/gofrs/uuid"
)

const (
	paddingHeaderLen = 1 + 2 + 2 // =5

	commandPaddingContinue byte = 0x00
	commandPaddingEnd      byte = 0x01
	commandPaddingDirect   byte = 0x02
)

func WriteWithPadding(buffer *buf.Buffer, p []byte, command byte, userUUID *uuid.UUID) {
	contentLen := int32(len(p))
	var paddingLen int32
	if contentLen < 900 {
		paddingLen = rand.Int31n(500) + 900 - contentLen
	}

	if userUUID != nil { // unnecessary, but keep the same with Xray
		buffer.Write(userUUID.Bytes())
	}
	buffer.WriteByte(command)
	binary.BigEndian.PutUint16(buffer.Extend(2), uint16(contentLen))
	binary.BigEndian.PutUint16(buffer.Extend(2), uint16(paddingLen))
	buffer.Write(p)
	buffer.Extend(int(paddingLen))
	log.Debugln("XTLS Vision write padding1: command=%v, payloadLen=%v, paddingLen=%v", command, contentLen, paddingLen)
}

func ApplyPadding(buffer *buf.Buffer, command byte, userUUID *uuid.UUID) {
	contentLen := int32(buffer.Len())
	var paddingLen int32
	if contentLen < 900 {
		paddingLen = rand.Int31n(500) + 900 - contentLen
	}

	binary.BigEndian.PutUint16(buffer.ExtendHeader(2), uint16(paddingLen))
	binary.BigEndian.PutUint16(buffer.ExtendHeader(2), uint16(contentLen))
	buffer.ExtendHeader(1)[0] = command
	if userUUID != nil { // unnecessary, but keep the same with Xray
		copy(buffer.ExtendHeader(uuid.Size), userUUID.Bytes())
	}
	buffer.Extend(int(paddingLen))
	log.Debugln("XTLS Vision write padding2: command=%d, payloadLen=%d, paddingLen=%d", command, contentLen, paddingLen)
}

func ReshapeBuffer(buffer *buf.Buffer) *buf.Buffer {
	if buffer.Len() <= buf.BufferSize-paddingHeaderLen {
		return nil
	}
	cutAt := bytes.LastIndex(buffer.Bytes(), tlsApplicationDataStart)
	if cutAt == -1 {
		cutAt = buf.BufferSize / 2
	}
	buffer2 := buf.New()
	buffer2.Write(buffer.From(cutAt))
	buffer.Truncate(cutAt)
	return buffer2
}
