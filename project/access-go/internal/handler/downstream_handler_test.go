package handler

import (
	"testing"

	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
	"sudooom.im.shared/proto"
)

func TestToFlatBufferAckStatus(t *testing.T) {
	tests := []struct {
		name   string
		status proto.MessageAckStatus
		want   im_protocol.AckStatus
	}{
		{name: "empty defaults to accepted", status: "", want: im_protocol.AckStatusACCEPTED},
		{name: "accepted", status: proto.MessageAckStatusAccepted, want: im_protocol.AckStatusACCEPTED},
		{name: "persisted", status: proto.MessageAckStatusPersisted, want: im_protocol.AckStatusPERSISTED},
		{name: "unknown", status: proto.MessageAckStatus("bad"), want: im_protocol.AckStatusUNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toFlatBufferAckStatus(tt.status); got != tt.want {
				t.Fatalf("toFlatBufferAckStatus(%q) = %s，期望 %s", tt.status, got.String(), tt.want.String())
			}
		})
	}
}
