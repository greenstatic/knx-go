// Package cemi provides the functionality to parse and generate KNX CEMI-encoded frames.
package cemi

import (
	"fmt"
	"io"

	"github.com/vapourismo/knx-go/knx/encoding"
	"github.com/vapourismo/knx-go/knx/util"
)

// MessageCode is used to identify the contents of a CEMI frame.
type MessageCode uint8

const (
	// LBusmonIndCode is the message code for L_Busmon.ind.
	LBusmonIndCode MessageCode = 0x2B

	// LDataReqCode is the message code for L_Data.req.
	LDataReqCode MessageCode = 0x11

	// LDataIndCode is the message code for L_Data.ind.
	LDataIndCode MessageCode = 0x29

	// LDataConCode is the message code for L_Data.con.
	LDataConCode MessageCode = 0x2E

	// LRawReqCode is the message code for L_Raw.req.
	LRawReqCode MessageCode = 0x10

	// LRawIndCode is the message code for L_Raw.ind.
	LRawIndCode MessageCode = 0x2D

	// LRawConCode is the message code for L_Raw.con.
	LRawConCode MessageCode = 0x2F

	// LPollDataReqCode MessageCode = 0x13
	// LPollDataConCode MessageCode = 0x25
)

// String converts the message code to a string.
func (code MessageCode) String() string {
	switch code {
	case LBusmonIndCode:
		return "LBusmonInd"

	case LDataReqCode:
		return "LDataReq"

	case LDataIndCode:
		return "LDataInd"

	case LDataConCode:
		return "LDataCon"

	case LRawReqCode:
		return "LRawReq"

	case LRawIndCode:
		return "LRawInd"

	case LRawConCode:
		return "LRawCon"

	default:
		return fmt.Sprintf("%#x", uint8(code))
	}
}

// Info is the additional info segment of a CEMI-encoded frame.
type Info []byte

// Size returns the packed size.
func (info Info) Size() uint {
	if len(info) > 255 {
		return 256
	}

	return 1 + uint(len(info))
}

// Pack the info structure into the buffer.
func (info Info) Pack(buffer []byte) {
	if len(info) > 255 {
		buffer[0] = 255
	} else {
		buffer[0] = byte(len(info))
	}

	copy(buffer[1:], info[:buffer[0]])
}

// Unpack initializes the structure by parsing the given data.
func (info *Info) Unpack(data []byte) (n uint, err error) {
	var length uint8

	n, err = util.Unpack(data, &length)
	if err != nil {
		return
	}

	if length > 0 {
		buf := make([]byte, length)
		n += uint(copy(buf, data[n:n+uint(length)]))
		*info = Info(buf)
	} else {
		*info = nil
	}

	return
}

// WriteTo writes an additional information segment.
func (info Info) WriteTo(w io.Writer) (int64, error) {
	length := uint8(len(info))
	return encoding.WriteSome(w, length, []byte(info[:length]))
}

// Message is the body of a CEMI-encoded frame.
type Message interface {
	io.WriterTo
	util.Packable
	MessageCode() MessageCode
}

// An UnsupportedMessage is the raw representation of a message inside a CEMI-encoded frame.
type UnsupportedMessage struct {
	Code MessageCode
	Data []byte
}

// Size returns the packed size.
func (body *UnsupportedMessage) Size() uint {
	return uint(len(body.Data))
}

// Pack the message body into the buffer.
func (body *UnsupportedMessage) Pack(buffer []byte) {
	copy(buffer, body.Data)
}

// MessageCode returns the message code.
func (body *UnsupportedMessage) MessageCode() MessageCode {
	return body.Code
}

// Unpack initializes the structure by parsing the given data.
func (body *UnsupportedMessage) Unpack(data []byte) (uint, error) {
	if len(body.Data) < len(data) {
		body.Data = make([]byte, len(data))
	}

	return uint(copy(body.Data, data)), nil
}

// WriteTo serializes the structure and writes it to the given Writer.
func (body *UnsupportedMessage) WriteTo(w io.Writer) (int64, error) {
	len, err := w.Write(body.Data)
	return int64(len), err
}

type messageUnpackable interface {
	util.Unpackable
	Message
}

// Unpack a message from a CEMI-encoded frame.
func Unpack(data []byte, message *Message) (n uint, err error) {
	var code MessageCode

	// Read header.
	n, err = util.Unpack(data, (*uint8)(&code))
	if err != nil {
		return
	}

	var body messageUnpackable

	// Decide which message is appropriate.
	switch code {
	case LBusmonIndCode:
		body = &LBusmonInd{}

	case LDataReqCode:
		body = &LDataReq{}

	case LDataConCode:
		body = &LDataCon{}

	case LDataIndCode:
		body = &LDataInd{}

	case LRawReqCode:
		body = &LRawReq{}

	case LRawConCode:
		body = &LRawCon{}

	case LRawIndCode:
		body = &LRawInd{}

	default:
		body = &UnsupportedMessage{Code: code}
	}

	// Parse the message.
	m, err := body.Unpack(data[n:])

	if err == nil {
		*message = body
	}

	return n + m, err
}

// Size returns the size for a CEMI-encoded frame with the given message.
func Size(message Message) uint {
	return 1 + message.Size()
}

// Pack assembles a CEMI-encoded frame using the given message.
func Pack(buffer []byte, message Message) {
	util.PackSome(buffer, uint8(message.MessageCode()), message)
}
