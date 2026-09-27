package main

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/url"
	"strconv"
	"time"
)

// Diagnostic subset: primary addressing, CI72, first instantaneous signed
// 16-bit flow-temperature record with VIF5A. Other encodings fail explicitly.
func readMBus(ctx context.Context, r ReadRequest) (any, error) {
	u, err := url.Parse(r.Connection)
	if err != nil {
		return nil, err
	}
	address, err := strconv.Atoi(r.Address)
	if err != nil || address < 1 || address > 250 {
		return nil, errors.New("M-Bus primary address must be 1..250")
	}
	c, err := (&net.Dialer{}).DialContext(ctx, "tcp", u.Host)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	deadline := time.Now().Add(8 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = c.SetDeadline(deadline)
	a := byte(address)
	if _, err = c.Write([]byte{0x10, 0x40, a, 0x40 + a, 0x16}); err != nil {
		return nil, err
	}
	ack := make([]byte, 1)
	if _, err = io.ReadFull(c, ack); err != nil {
		return nil, err
	}
	if ack[0] != 0xe5 {
		return nil, errors.New("M-Bus initialization not acknowledged")
	}
	if _, err = c.Write([]byte{0x10, 0x7b, a, 0x7b + a, 0x16}); err != nil {
		return nil, err
	}
	header := make([]byte, 4)
	if _, err = io.ReadFull(c, header); err != nil {
		return nil, err
	}
	if header[0] != 0x68 || header[3] != 0x68 || header[1] != header[2] || header[1] < 15 {
		return nil, errors.New("invalid M-Bus frame header")
	}
	tail := make([]byte, int(header[1])+2)
	if _, err = io.ReadFull(c, tail); err != nil {
		return nil, err
	}
	return decodeMBusTemperature(tail, a)
}
func decodeMBusTemperature(tail []byte, address byte) (any, error) {
	if len(tail) < 17 {
		return nil, errors.New("short M-Bus frame")
	}
	body := tail[:len(tail)-2]
	var sum byte
	for _, b := range body {
		sum += b
	}
	if sum != tail[len(tail)-2] || tail[len(tail)-1] != 0x16 || body[0]&0xcf != 8 || body[1] != address || body[2] != 0x72 {
		return nil, errors.New("invalid M-Bus checksum, address or response")
	}
	if body[12] != 0 {
		return nil, errors.New("M-Bus meter reports a status error")
	}
	// No guessed scaling for vendor-specific or extended records.
	if len(body) != 19 || body[15] != 2 || body[16] != 0x5a {
		return nil, errors.New("supported subset: one CI72 DIF02 VIF5A flow-temperature record; other meters need a gateway mapping")
	}
	return float64(int16(binary.LittleEndian.Uint16(body[17:19]))) / 10, nil
}
