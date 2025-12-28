package server

import (
	"encoding/binary"
	"io"
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
)

// TestBuildAuthRequest 测试构建认证请求
func TestBuildAuthRequest(t *testing.T) {
	token := "test-token-123"
	deviceID := "device-001"
	platform := im_protocol.PlatformWEB
	appVersion := "1.0.0"

	// 构建认证请求
	builder := flatbuffers.NewBuilder(256)

	tokenOffset := builder.CreateString(token)
	deviceIDOffset := builder.CreateString(deviceID)
	appVersionOffset := builder.CreateString(appVersion)

	im_protocol.AuthRequestStart(builder)
	im_protocol.AuthRequestAddToken(builder, tokenOffset)
	im_protocol.AuthRequestAddDeviceId(builder, deviceIDOffset)
	im_protocol.AuthRequestAddPlatform(builder, platform)
	im_protocol.AuthRequestAddAppVersion(builder, appVersionOffset)
	authReqOffset := im_protocol.AuthRequestEnd(builder)

	builder.Finish(authReqOffset)
	authReqBytes := builder.FinishedBytes()

	// 验证构建的数据
	if len(authReqBytes) == 0 {
		t.Fatal("认证请求构建失败，长度为 0")
	}

	// 解析验证
	authReq := im_protocol.GetRootAsAuthRequest(authReqBytes, 0)

	if string(authReq.Token()) != token {
		t.Errorf("token 不匹配，期望: %s, 实际: %s", token, string(authReq.Token()))
	}

	if string(authReq.DeviceId()) != deviceID {
		t.Errorf("deviceID 不匹配，期望: %s, 实际: %s", deviceID, string(authReq.DeviceId()))
	}

	if authReq.Platform() != platform {
		t.Errorf("platform 不匹配，期望: %s, 实际: %s", platform.String(), authReq.Platform().String())
	}

	if string(authReq.AppVersion()) != appVersion {
		t.Errorf("appVersion 不匹配，期望: %s, 实际: %s", appVersion, string(authReq.AppVersion()))
	}

	t.Logf("认证请求构建成功，大小: %d bytes", len(authReqBytes))
}

// TestBuildAuthFrame 测试构建认证帧
func TestBuildAuthFrame(t *testing.T) {
	// 构建认证请求
	builder := flatbuffers.NewBuilder(256)

	tokenOffset := builder.CreateString("test-token")
	deviceIDOffset := builder.CreateString("device-001")
	appVersionOffset := builder.CreateString("1.0.0")

	im_protocol.AuthRequestStart(builder)
	im_protocol.AuthRequestAddToken(builder, tokenOffset)
	im_protocol.AuthRequestAddDeviceId(builder, deviceIDOffset)
	im_protocol.AuthRequestAddPlatform(builder, im_protocol.PlatformWEB)
	im_protocol.AuthRequestAddAppVersion(builder, appVersionOffset)
	authReqOffset := im_protocol.AuthRequestEnd(builder)

	builder.Finish(authReqOffset)
	authReqBytes := builder.FinishedBytes()

	// 构建帧：header + body
	const frameHeaderSize = 5
	const frameTypeAuth = byte(1)

	frame := make([]byte, frameHeaderSize+len(authReqBytes))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(authReqBytes)))
	frame[4] = frameTypeAuth
	copy(frame[frameHeaderSize:], authReqBytes)

	// 验证帧结构
	if len(frame) != frameHeaderSize+len(authReqBytes) {
		t.Fatalf("帧长度不正确，期望: %d, 实际: %d",
			frameHeaderSize+len(authReqBytes), len(frame))
	}

	// 验证帧头
	length := binary.BigEndian.Uint32(frame[:4])
	if length != uint32(len(authReqBytes)) {
		t.Errorf("帧头长度字段不正确，期望: %d, 实际: %d", len(authReqBytes), length)
	}

	frameType := frame[4]
	if frameType != frameTypeAuth {
		t.Errorf("帧类型不正确，期望: %d, 实际: %d", frameTypeAuth, frameType)
	}

	// 验证帧体
	frameBody := frame[frameHeaderSize:]
	if len(frameBody) != len(authReqBytes) {
		t.Errorf("帧体长度不正确，期望: %d, 实际: %d", len(authReqBytes), len(frameBody))
	}

	// 解析帧体并验证
	authReq := im_protocol.GetRootAsAuthRequest(frameBody, 0)
	if string(authReq.Token()) != "test-token" {
		t.Errorf("从帧中解析的 token 不正确")
	}

	t.Logf("认证帧构建成功，总大小: %d bytes (header: %d, body: %d)",
		len(frame), frameHeaderSize, len(authReqBytes))
}

// TestParseAuthResponse 测试解析认证响应
func TestParseAuthResponse(t *testing.T) {
	// 构建一个模拟的成功响应
	builder := flatbuffers.NewBuilder(256)

	reqIDOffset := builder.CreateString("")
	msgOffset := builder.CreateString("success")

	im_protocol.ClientResponseStart(builder)
	im_protocol.ClientResponseAddReqId(builder, reqIDOffset)
	im_protocol.ClientResponseAddTimestamp(builder, 1234567890)
	im_protocol.ClientResponseAddCode(builder, im_protocol.ErrorCodeSUCCESS)
	im_protocol.ClientResponseAddMsg(builder, msgOffset)
	im_protocol.ClientResponseAddPayloadType(builder, im_protocol.ResponsePayloadNONE)
	respOffset := im_protocol.ClientResponseEnd(builder)

	builder.Finish(respOffset)
	respBytes := builder.FinishedBytes()

	// 解析响应
	response := im_protocol.GetRootAsClientResponse(respBytes, 0)

	// 验证响应
	if response.Code() != im_protocol.ErrorCodeSUCCESS {
		t.Errorf("错误码不匹配，期望: %s, 实际: %s",
			im_protocol.ErrorCodeSUCCESS.String(), response.Code().String())
	}

	if string(response.Msg()) != "success" {
		t.Errorf("消息不匹配，期望: success, 实际: %s", string(response.Msg()))
	}

	if response.PayloadType() != im_protocol.ResponsePayloadNONE {
		t.Errorf("payload 类型不匹配")
	}

	t.Logf("成功解析认证响应，错误码: %s, 消息: %s",
		response.Code().String(), string(response.Msg()))
}

// mockStream 用于测试的模拟流
type mockStream struct {
	readBuffer  []byte
	writeBuffer []byte
	readPos     int
}

func (m *mockStream) Read(p []byte) (n int, err error) {
	if m.readPos >= len(m.readBuffer) {
		return 0, io.EOF
	}

	n = copy(p, m.readBuffer[m.readPos:])
	m.readPos += n
	return n, nil
}

func (m *mockStream) Write(p []byte) (n int, err error) {
	m.writeBuffer = append(m.writeBuffer, p...)
	return len(p), nil
}

// TestAuthFrameReadWrite 测试认证帧的读写
func TestAuthFrameReadWrite(t *testing.T) {
	// 构建认证请求
	builder := flatbuffers.NewBuilder(256)

	tokenOffset := builder.CreateString("test-token-xyz")
	deviceIDOffset := builder.CreateString("device-002")
	appVersionOffset := builder.CreateString("2.0.0")

	im_protocol.AuthRequestStart(builder)
	im_protocol.AuthRequestAddToken(builder, tokenOffset)
	im_protocol.AuthRequestAddDeviceId(builder, deviceIDOffset)
	im_protocol.AuthRequestAddPlatform(builder, im_protocol.PlatformDESKTOP)
	im_protocol.AuthRequestAddAppVersion(builder, appVersionOffset)
	authReqOffset := im_protocol.AuthRequestEnd(builder)

	builder.Finish(authReqOffset)
	authReqBytes := builder.FinishedBytes()

	// 构建帧
	const frameHeaderSize = 5
	const frameTypeAuth = byte(1)

	frame := make([]byte, frameHeaderSize+len(authReqBytes))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(authReqBytes)))
	frame[4] = frameTypeAuth
	copy(frame[frameHeaderSize:], authReqBytes)

	// 使用 mockStream 测试写入
	stream := &mockStream{}
	_, err := stream.Write(frame)
	if err != nil {
		t.Fatalf("写入帧失败: %v", err)
	}

	// 验证写入的数据
	if len(stream.writeBuffer) != len(frame) {
		t.Fatalf("写入的数据长度不正确，期望: %d, 实际: %d",
			len(frame), len(stream.writeBuffer))
	}

	// 使用相同的数据创建读取流
	readStream := &mockStream{readBuffer: stream.writeBuffer}

	// 读取帧头
	header := make([]byte, frameHeaderSize)
	_, err = io.ReadFull(readStream, header)
	if err != nil {
		t.Fatalf("读取帧头失败: %v", err)
	}

	length := binary.BigEndian.Uint32(header[:4])
	frameType := header[4]

	if length != uint32(len(authReqBytes)) {
		t.Errorf("读取的长度不正确，期望: %d, 实际: %d", len(authReqBytes), length)
	}

	if frameType != frameTypeAuth {
		t.Errorf("读取的帧类型不正确，期望: %d, 实际: %d", frameTypeAuth, frameType)
	}

	// 读取帧体
	body := make([]byte, length)
	_, err = io.ReadFull(readStream, body)
	if err != nil {
		t.Fatalf("读取帧体失败: %v", err)
	}

	// 解析并验证
	authReq := im_protocol.GetRootAsAuthRequest(body, 0)

	if string(authReq.Token()) != "test-token-xyz" {
		t.Errorf("token 不匹配")
	}

	if string(authReq.DeviceId()) != "device-002" {
		t.Errorf("deviceID 不匹配")
	}

	if authReq.Platform() != im_protocol.PlatformDESKTOP {
		t.Errorf("platform 不匹配")
	}

	t.Logf("认证帧读写测试成功")
}
