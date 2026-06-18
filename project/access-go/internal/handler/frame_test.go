package handler

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestReadFrameReadsHeaderAndBody(t *testing.T) {
	body := []byte("hello")
	reader := bytes.NewReader(buildTestFrame(FrameTypeRequest, body))

	frameType, gotBody, err := readFrame(reader, 1024)
	if err != nil {
		t.Fatalf("读取帧失败: %v", err)
	}
	if frameType != FrameTypeRequest {
		t.Fatalf("帧类型不匹配，期望 %d，实际 %d", FrameTypeRequest, frameType)
	}
	if !bytes.Equal(gotBody, body) {
		t.Fatalf("帧体不匹配，期望 %q，实际 %q", body, gotBody)
	}
}

func TestReadFrameRejectsBodyOverLimitWithoutReadingBody(t *testing.T) {
	body := []byte("oversized")
	reader := bytes.NewReader(buildTestFrame(FrameTypeAuth, body))

	frameType, gotBody, err := readFrame(reader, uint32(len(body)-1))
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("超过限制时应返回 ErrFrameTooLarge，实际: %v", err)
	}
	if frameType != FrameTypeAuth {
		t.Fatalf("超过限制时仍应返回已解析的帧类型，期望 %d，实际 %d", FrameTypeAuth, frameType)
	}
	if gotBody != nil {
		t.Fatalf("超过限制时不应分配或读取帧体，实际长度: %d", len(gotBody))
	}
	if reader.Len() != len(body) {
		t.Fatalf("超过限制时不应继续读取帧体，剩余 %d，期望 %d", reader.Len(), len(body))
	}
}

func TestReadFrameReturnsErrorForIncompleteHeader(t *testing.T) {
	_, _, err := readFrame(bytes.NewReader([]byte{0x01, 0x02}), 1024)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("帧头不完整应返回 ErrUnexpectedEOF，实际: %v", err)
	}
}

func TestNormalizeMaxFrameSizeDefaultsToBoundedValue(t *testing.T) {
	if got := normalizeMaxFrameSize(0); got != defaultMaxFrameSize {
		t.Fatalf("maxFrameSize 为 0 时应使用有界默认值，期望 %d，实际 %d", defaultMaxFrameSize, got)
	}
}

func buildTestFrame(frameType byte, body []byte) []byte {
	frame := make([]byte, FrameHeaderSize+len(body))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(body)))
	frame[4] = frameType
	copy(frame[FrameHeaderSize:], body)
	return frame
}
