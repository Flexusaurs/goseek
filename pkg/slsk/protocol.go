package slsk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type MsgType int32

const (
	// Basic control / discovery
	MsgHello       MsgType = 1  // peer hello (sends username or id)
	MsgRoomsList   MsgType = 10 // list of room names
	MsgJoinRoom    MsgType = 11 // request join room
	MsgRoomMembers MsgType = 12 // list of members in a room

	// Files / transfers
	MsgFileList     MsgType = 20 // request/announce file list
	MsgFileAdvert   MsgType = 21 // advertise a single file (fileID, name, size)
	MsgGetFile      MsgType = 30 // request a file (transfer initiation)
	MsgFileChunk    MsgType = 31 // chunk frame for file transfer
	MsgTransferInit MsgType = 50 // transfer negotiation/init

	// Chat
	MsgChat MsgType = 40

	// misc
	MsgPing MsgType = 99
)

var MsgNames = map[MsgType]string{
	MsgHello:        "HELLO",
	MsgRoomsList:    "ROOMS_LIST",
	MsgJoinRoom:     "JOIN_ROOM",
	MsgRoomMembers:  "ROOM_MEMBERS",
	MsgFileList:     "FILE_LIST",
	MsgFileAdvert:   "FILE_ADVERT",
	MsgGetFile:      "GET_FILE",
	MsgFileChunk:    "FILE_CHUNK",
	MsgTransferInit: "TRANSFER_INIT",
	MsgChat:         "CHAT",
	MsgPing:         "PING",
}

func (m MsgType) String() string {
	if s, ok := MsgNames[m]; ok {
		return s
	}
	return fmt.Sprintf("Msg(%d)", int32(m))
}

type ErrorCode int32

const (
	ErrNone       ErrorCode = 0
	ErrInvalid    ErrorCode = 1
	ErrNotFound   ErrorCode = 2
	ErrPermission ErrorCode = 3
	ErrInternal   ErrorCode = 10
)

var ErrorMessages = map[ErrorCode]string{
	ErrNone:       "no error",
	ErrInvalid:    "invalid request",
	ErrNotFound:   "not found",
	ErrPermission: "permission denied",
	ErrInternal:   "internal error",
}

func (e ErrorCode) String() string {
	if s, ok := ErrorMessages[e]; ok {
		return s
	}

	return fmt.Sprintf("ErrorCode(%d)", int32(e))
}

func PackMessage(msg MsgType, payload []byte) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int32(msg))
	if len(payload) > 0 {
		_, _ = buf.Write(payload)
	}
	return prefixLength(buf.Bytes())
}

func ReadMessage(r io.Reader) (MsgType, []byte, error) {
	length, err := readInt32(r)
	if err != nil {
		return 0, nil, err
	}
	// length includes msgID (4 bytes) + payload
	msgID, err := readInt32(r)
	if err != nil {
		return 0, nil, err
	}
	remaining := int(length) - 4 // subtract msgID
	if remaining < 0 {
		return 0, nil, fmt.Errorf("invalid message length: %d", length)
	}
	var payload []byte
	if remaining > 0 {
		payload = make([]byte, remaining)
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, err
		}
	}
	return MsgType(msgID), payload, nil
}

func prefixLength(data []byte) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int32(len(data)))
	_, _ = buf.Write(data) // <- make sure no stray brackets here
	return buf.Bytes()
}

//utils - reading and writing buf ptrs

func writeInt32(buf *bytes.Buffer, n int32) error {
	return binary.Write(buf, binary.LittleEndian, n)
}

func readInt32(r io.Reader) (int32, error) {
	var n int32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return 0, err
	}
	return n, nil
}

func writeInt64(buf *bytes.Buffer, n int64) error {
	return binary.Write(buf, binary.LittleEndian, n)
}

func readInt64(r io.Reader) (int64, error) {
	var n int64
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return 0, err
	}
	return n, nil
}

//string encodings

const maxStringLen = 50_000_000 //50mb string

func WriteString(buf *bytes.Buffer, s string) error {
	if int64(len(s)) > maxStringLen {
		return fmt.Errorf("String too large, throwing input: %d", len(s))
	}

	if err := writeInt32(buf, int32(len(s))); err != nil {
		return err
	}

	_, err := buf.WriteString(s)
	return err
}

func ReadString(r io.Reader) (string, error) {
	l, err := readInt32(r)
	if err != nil {
		return "", err
	}
	if l < 0 || int64(l) > maxStringLen {
		return "", fmt.Errorf("invalid string length %d", l)
	}
	if l == 0 {
		return "", nil
	}
	b := make([]byte, l)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

//message encoders and decoders

func EncodeHello(username string) []byte {
	buf := new(bytes.Buffer)
	_ = WriteString(buf, username)
	return PackMessage(MsgHello, buf.Bytes())
}

func DecodeHello(payload []byte) (string, error) {
	r := bytes.NewReader(payload)
	return ReadString(r)
}

//ping has no payload

func EncodePing() []byte {
	return PackMessage(MsgPing, nil)
}

func EncodeFileAdvert(fileID, name string, size int64) []byte {
	buf := new(bytes.Buffer)
	_ = WriteString(buf, fileID)
	_ = WriteString(buf, name)
	_ = writeInt64(buf, size)

	return PackMessage(MsgFileAdvert, buf.Bytes())
}

func DecodeFileAdvert(payload []byte) (fileID, name string, size int64, err error) {
	r := bytes.NewReader(payload)
	fileID, err = ReadString(r)
	if err != nil {
		return "", "", 0, err
	}
	name, err = ReadString(r)
	if err != nil {
		return "", "", 0, err
	}
	size, err = readInt64(r)
	if err != nil {
		return "", "", 0, err
	}
	return fileID, name, size, nil
}

// GetFile: payload = [string transferID][string fileID][int64 offset]
// transferID is a client-local ID used to match transfer replies/chunks.
func EncodeGetFile(transferID, fileID string, offset int64) []byte {
	buf := new(bytes.Buffer)
	_ = WriteString(buf, transferID)
	_ = WriteString(buf, fileID)
	_ = writeInt64(buf, offset)
	return PackMessage(MsgGetFile, buf.Bytes())
}

// DecodeGetFile decodes GetFile payload.
func DecodeGetFile(payload []byte) (transferID string, fileID string, offset int64, err error) {
	r := bytes.NewReader(payload)
	transferID, err = ReadString(r)
	if err != nil {
		return "", "", 0, err
	}
	fileID, err = ReadString(r)
	if err != nil {
		return "", "", 0, err
	}
	offset, err = readInt64(r)
	if err != nil {
		return "", "", 0, err
	}
	return transferID, fileID, offset, nil
}

func EncodeFileChunk(transferID string, offset int64, chunk []byte) []byte {
	buf := new(bytes.Buffer)
	_ = WriteString(buf, transferID)
	_ = writeInt64(buf, offset)
	_ = writeInt32(buf, int32(len(chunk)))
	if len(chunk) > 0 {
		_, _ = buf.Write(chunk)
	}
	return PackMessage(MsgFileChunk, buf.Bytes())
}

func DecodeFileChunk(payload []byte) (transferID string, offset int64, chunk []byte, err error) {
	r := bytes.NewReader(payload)
	transferID, err = ReadString(r)
	if err != nil {
		return "", 0, nil, err
	}
	offset, err = readInt64(r)
	if err != nil {
		return "", 0, nil, err
	}
	chLen, err := readInt32(r)
	if err != nil {
		return "", 0, nil, err
	}
	if chLen < 0 || chLen > 100*1024*1024 {
		return "", 0, nil, fmt.Errorf("invalid chunk length %d", chLen)
	}
	if chLen == 0 {
		return transferID, offset, nil, nil
	}
	chunk = make([]byte, chLen)
	if _, err := io.ReadFull(r, chunk); err != nil {
		return "", 0, nil, err
	}
	return transferID, offset, chunk, nil
}

// ErrUnexpectedEOF is returned when a read produced EOF unexpectedly.
var ErrUnexpectedEOF = errors.New("unexpected EOF while decoding")
