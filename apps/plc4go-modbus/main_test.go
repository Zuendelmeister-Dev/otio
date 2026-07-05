package main

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestScaled(t *testing.T) {
	if got := scaled(8.25, 0.01); got != 825 {
		t.Fatalf("expected scaled value 825, got %d", got)
	}
}

func TestRegisterBankRejectsInvalidRange(t *testing.T) {
	bank := &registerBank{}
	bank.set([]uint16{1, 2, 3})
	if _, ok := bank.read(99, 2); ok {
		t.Fatal("expected out-of-range read to fail")
	}
	if _, ok := bank.read(0, 0); ok {
		t.Fatal("expected zero-count read to fail")
	}
}

func TestHandleModbusConnectionReadsHoldingRegister(t *testing.T) {
	bank := &registerBank{}
	bank.set([]uint16{825})

	server, client := net.Pipe()
	defer client.Close()
	done := make(chan struct{})
	go func() {
		handleModbusConnection(server, bank)
		close(done)
	}()

	request := make([]byte, 12)
	binary.BigEndian.PutUint16(request[0:2], 0x1234)
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint16(request[4:6], 6)
	request[6] = 1
	request[7] = 3
	binary.BigEndian.PutUint16(request[8:10], 0)
	binary.BigEndian.PutUint16(request[10:12], 1)

	_ = client.SetDeadline(time.Now().Add(time.Second))
	if _, err := client.Write(request); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, 11)
	if _, err := io.ReadFull(client, response); err != nil {
		t.Fatal(err)
	}

	if got := binary.BigEndian.Uint16(response[0:2]); got != 0x1234 {
		t.Fatalf("transaction id = %#x, want 0x1234", got)
	}
	if response[7] != 3 || response[8] != 2 {
		t.Fatalf("unexpected Modbus response PDU: %#v", response[7:])
	}
	if got := binary.BigEndian.Uint16(response[9:11]); got != 825 {
		t.Fatalf("register value = %d, want 825", got)
	}

	client.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("server connection did not close")
	}
}
