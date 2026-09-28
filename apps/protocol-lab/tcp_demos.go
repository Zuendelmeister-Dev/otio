package main

// Small read-only wire fixtures, not PLC or meter emulators. S7 exposes DB1
// bytes 0..3 as REAL; M-Bus exposes one flow-temperature record, primary 1.
import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"time"
)

func listenTCPDemo(ctx context.Context, sim *simulator, protocol string) (net.Listener, error) {
	port := 1102
	if protocol == "mbus-tcp" {
		port = 1504
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	go serveTCPDemo(ctx, l, sim, protocol)
	return l, nil
}
func serveTCPDemo(ctx context.Context, l net.Listener, sim *simulator, protocol string) {
	stop := context.AfterFunc(ctx, func() { _ = l.Close() })
	defer stop()
	slots := make(chan struct{}, 16)
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		select {
		case slots <- struct{}{}:
		default:
			_ = c.Close()
			continue
		}
		go func() {
			defer func() { <-slots }()
			defer c.Close()
			stop := context.AfterFunc(ctx, func() { _ = c.Close() })
			defer stop()
			for {
				_ = c.SetDeadline(time.Now().Add(10 * time.Second))
				var response []byte
				if protocol == "s7" {
					header := make([]byte, 4)
					if _, err := io.ReadFull(c, header); err != nil {
						return
					}
					n := int(binary.BigEndian.Uint16(header[2:]))
					if header[0] != 3 || header[1] != 0 || n < 7 || n > 4096 {
						return
					}
					body := make([]byte, n-4)
					if _, err := io.ReadFull(c, body); err != nil {
						return
					}
					response = s7DemoResponse(body, sim.snapshot().Temperature)
					if response == nil {
						return
					}
					response = append([]byte{3, 0, byte((len(response) + 4) >> 8), byte(len(response) + 4)}, response...)
				} else {
					request := make([]byte, 5)
					if _, err := io.ReadFull(c, request); err != nil {
						return
					}
					if request[0] != 0x10 || request[4] != 0x16 || request[3] != request[1]+request[2] || request[2] != 1 {
						return
					}
					switch request[1] {
					case 0x40:
						response = []byte{0xe5}
					case 0x5b, 0x7b:
						response = mbusDemoFrame(sim.snapshot().Temperature)
					default:
						return
					}
				}
				if _, err := c.Write(response); err != nil {
					return
				}
			}
		}()
	}
}
func s7DemoResponse(body []byte, temperature float64) []byte {
	if len(body) >= 7 && body[1] == 0xe0 {
		response := append([]byte(nil), body...)
		response[1] = 0xd0
		copy(response[2:4], body[4:6])
		response[4] = 0
		response[5] = 1
		return response
	}
	if len(body) < 13 || body[0] != 2 || body[1] != 0xf0 || body[2]&0x80 == 0 {
		return nil
	}
	req := body[3:]
	if req[0] != 0x32 || req[1] != 1 {
		return nil
	}
	plen, dlen := int(binary.BigEndian.Uint16(req[6:8])), int(binary.BigEndian.Uint16(req[8:10]))
	if plen < 1 || len(req) != 10+plen+dlen {
		return nil
	}
	params := req[10 : 10+plen]
	var out, data []byte
	switch params[0] {
	case 0xf0:
		if plen != 8 {
			return nil
		}
		out = []byte{0xf0, 0, 0, 1, 0, 1, 1, 0xe0}
	case 4:
		if plen < 2 || params[1] == 0 || plen != 2+int(params[1])*12 {
			return nil
		}
		out = []byte{4, params[1]}
		raw := make([]byte, 4)
		binary.BigEndian.PutUint32(raw, math.Float32bits(float32(temperature)))
		for i := 0; i < int(params[1]); i++ {
			p := params[2+i*12 : 14+i*12]
			count := int(binary.BigEndian.Uint16(p[4:6]))
			offset := (int(p[9])<<16 | int(p[10])<<8 | int(p[11]))
			size := count
			if p[3] == 8 {
				size = count * 4
			}
			valid := p[0] == 0x12 && p[1] == 10 && p[2] == 0x10 && (p[3] == 2 || p[3] == 8) && p[6] == 0 && p[7] == 1 && p[8] == 0x84 && offset%8 == 0 && size > 0 && offset/8+size <= 4
			if !valid {
				data = append(data, 5, 0, 0, 0)
				continue
			}
			data = append(data, 0xff, 4, byte(size*8>>8), byte(size*8))
			data = append(data, raw[offset/8:offset/8+size]...)
			if size%2 != 0 && i < int(params[1])-1 {
				data = append(data, 0)
			}
		}
	default:
		return nil
	}
	response := []byte{2, 0xf0, 0x80, 0x32, 3, 0, 0, req[4], req[5], byte(len(out) >> 8), byte(len(out)), byte(len(data) >> 8), byte(len(data)), 0, 0}
	return append(append(response, out...), data...)
}
func mbusDemoFrame(temperature float64) []byte {
	// CI 72, 12-byte fixed header, DIF 02 signed 16-bit, VIF 5A: 0.1 °C.
	body := []byte{8, 1, 0x72, 1, 0, 0, 0, 0x49, 0x6a, 1, 4, 0, 0, 0, 0, 2, 0x5a}
	v := int16(math.Round(temperature * 10))
	body = append(body, byte(v), byte(v>>8))
	var sum byte
	for _, b := range body {
		sum += b
	}
	return append(append([]byte{0x68, byte(len(body)), byte(len(body)), 0x68}, body...), sum, 0x16)
}
