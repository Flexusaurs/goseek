package tests

import (
	"bytes"
	"goseek/pkg/slsk"
	"testing"
)

// helper to roundtrip encode/decode and ensuring msg type and payload consistency
func roundTrip(t *testing.T, msg slsk.MsgType, payload []byte) {
	t.Helper()
	packet := slsk.PackMessage(msg, payload)
	msgOut, payloadOut, err := slsk.ReadMessage(bytes.NewReader(packet))
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	if msgOut != msg {
		t.Fatalf("msg type mismatch: got %v, want %v", msgOut, msg)
	}
	if !bytes.Equal(payload, payloadOut) {
		t.Fatalf("payload mismatch: got=%v want=%v", payloadOut, payload)
	}
}

func TestPackAndReadMessage(t *testing.T) {
	payload := []byte("hello world")
	roundTrip(t, slsk.MsgHello, payload)
}

func TestStringEncoding(t *testing.T) {
	buf := new(bytes.Buffer)
	err := slsk.WriteString(buf, "test123")
	if err != nil {
		t.Fatalf("WriteString failed: %v", err)
	}

	r := bytes.NewReader(buf.Bytes())
	s, err := slsk.ReadString(r)
	if err != nil {
		t.Fatalf("ReadString failed: %v", err)
	}
	if s != "test123" {
		t.Fatalf("expected 'test123', got '%s'", s)
	}
}

func TestHelloEncoding(t *testing.T) {
	pkt := slsk.EncodeHello("myuser")
	msg, payload, err := slsk.ReadMessage(bytes.NewReader(pkt))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if msg != slsk.MsgHello {
		t.Fatalf("expected MsgHello, got %v", msg)
	}
	username, err := slsk.DecodeHello(payload)
	if err != nil {
		t.Fatalf("DecodeHello: %v", err)
	}
	if username != "myuser" {
		t.Fatalf("expected username 'myuser', got '%s'", username)
	}
}

func TestFileAdvertEncoding(t *testing.T) {
	packet := slsk.EncodeFileAdvert("id123", "song.mp3", 123456)
	msg, payload, err := slsk.ReadMessage(bytes.NewReader(packet))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if msg != slsk.MsgFileAdvert {
		t.Fatalf("expected MsgFileAdvert, got %v", msg)
	}

	fileID, name, size, err := slsk.DecodeFileAdvert(payload)
	if err != nil {
		t.Fatalf("DecodeFileAdvert failed: %v", err)
	}
	if fileID != "id123" || name != "song.mp3" || size != 123456 {
		t.Fatalf("decoded mismatch: got (%s,%s,%d)", fileID, name, size)
	}
}

func TestGetFileEncoding(t *testing.T) {
	packet := slsk.EncodeGetFile("tx1", "file123", 2048)
	msg, payload, err := slsk.ReadMessage(bytes.NewReader(packet))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if msg != slsk.MsgGetFile {
		t.Fatalf("expected MsgGetFile, got %v", msg)
	}

	txID, fileID, offset, err := slsk.DecodeGetFile(payload)
	if err != nil {
		t.Fatalf("DecodeGetFile: %v", err)
	}
	if txID != "tx1" || fileID != "file123" || offset != 2048 {
		t.Fatalf("decoded mismatch: got (%s,%s,%d)", txID, fileID, offset)
	}
}

func TestFileChunkEncoding(t *testing.T) {
	chunk := []byte("abcdefg")
	packet := slsk.EncodeFileChunk("tx1", 512, chunk)

	msg, payload, err := slsk.ReadMessage(bytes.NewReader(packet))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if msg != slsk.MsgFileChunk {
		t.Fatalf("expected MsgFileChunk, got %v", msg)
	}

	txID, offset, data, err := slsk.DecodeFileChunk(payload)
	if err != nil {
		t.Fatalf("DecodeFileChunk: %v", err)
	}
	if txID != "tx1" || offset != 512 || !bytes.Equal(chunk, data) {
		t.Fatalf("decoded mismatch: got (%s,%d,%v)", txID, offset, data)
	}
}

func TestMalformedMessage(t *testing.T) {
	// intentionally short packet
	data := []byte{0x01, 0x00, 0x00}
	_, _, err := slsk.ReadMessage(bytes.NewReader(data))
	if err == nil {
		t.Fatalf("expected error for malformed message")
	}
}
