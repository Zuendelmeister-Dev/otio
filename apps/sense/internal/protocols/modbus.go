package protocols

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

func ReadHoldingRegister(host string, port int, unitID byte, register uint16) (uint16, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 2*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	transactionID := uint16(time.Now().UnixNano() & 0xffff)
	request := make([]byte, 12)
	binary.BigEndian.PutUint16(request[0:2], transactionID)
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint16(request[4:6], 6)
	request[6] = unitID
	request[7] = 3
	binary.BigEndian.PutUint16(request[8:10], register)
	binary.BigEndian.PutUint16(request[10:12], 1)
	if _, err := conn.Write(request); err != nil {
		return 0, err
	}
	header := make([]byte, 7)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, err
	}
	length := binary.BigEndian.Uint16(header[4:6])
	if binary.BigEndian.Uint16(header[0:2]) != transactionID || binary.BigEndian.Uint16(header[2:4]) != 0 || header[6] != unitID {
		return 0, errors.New("Modbus response does not match request")
	}
	if length != 3 && length != 5 {
		return 0, errors.New("invalid Modbus response length")
	}
	pdu := make([]byte, int(length)-1)
	if _, err := io.ReadFull(conn, pdu); err != nil {
		return 0, err
	}
	if pdu[0] == 0x83 && len(pdu) == 2 {
		return 0, fmt.Errorf("Modbus exception code %d", pdu[1])
	}
	if pdu[0] != 3 || pdu[1] != 2 || len(pdu) != 4 {
		return 0, errors.New("invalid Modbus read response")
	}
	return binary.BigEndian.Uint16(pdu[2:4]), nil
}
