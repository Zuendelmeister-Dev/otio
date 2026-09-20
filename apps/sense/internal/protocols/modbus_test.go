package protocols

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestReadHoldingRegister(t *testing.T) {
	for _, name := range []string{"valid", "transaction", "protocol", "unit", "length", "function", "byte count", "exception", "truncated"} {
		t.Run(name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan struct{})
			go func() {
				defer close(done)
				conn, err := listener.Accept()
				if err != nil {
					t.Error(err)
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				request := make([]byte, 12)
				if _, err := io.ReadFull(conn, request); err != nil {
					t.Error(err)
					return
				}
				if request[6] != 7 || request[7] != 3 || binary.BigEndian.Uint16(request[8:10]) != 123 || binary.BigEndian.Uint16(request[10:12]) != 1 {
					t.Errorf("invalid request: %v", request)
				}
				response := append(append([]byte{}, request[:7]...), 3, 2, 0x12, 0x34)
				binary.BigEndian.PutUint16(response[4:6], 5)
				switch name {
				case "transaction":
					response[0] ^= 1
				case "protocol":
					response[3] = 1
				case "unit":
					response[6]++
				case "length":
					binary.BigEndian.PutUint16(response[4:6], 65535)
				case "function":
					response[7] = 4
				case "byte count":
					response[8] = 1
				case "exception":
					response[5], response[7], response[8] = 3, 0x83, 2
					response = response[:9]
				case "truncated":
					response = response[:10]
				}
				_, _ = conn.Write(response)
			}()
			value, err := ReadHoldingRegister("127.0.0.1", listener.Addr().(*net.TCPAddr).Port, 7, 123)
			<-done
			if name == "valid" {
				if err != nil || value != 0x1234 {
					t.Fatalf("got %x, %v", value, err)
				}
			} else if err == nil {
				t.Fatalf("accepted invalid %s response", name)
			}
		})
	}
}
