package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

type simValues struct {
	Temperature float64 `json:"temperature"`
	Running     bool    `json:"running"`
	Mode        string  `json:"mode"`
}
type simulator struct {
	mu         sync.RWMutex
	values     simValues
	started    time.Time
	publishers map[string]string
}

func newSimulator() *simulator {
	return &simulator{values: simValues{Temperature: 23.5, Running: true, Mode: "constant"}, started: time.Now(), publishers: map[string]string{}}
}
func (s *simulator) snapshot() simValues {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.values
	if v.Mode == "sine" {
		v.Temperature = math.Max(0, math.Min(650, v.Temperature+2*math.Sin(time.Since(s.started).Seconds()/5)))
	}
	return v
}
func (s *simulator) update(v simValues) error {
	if math.IsNaN(v.Temperature) || math.IsInf(v.Temperature, 0) || v.Temperature < 0 || v.Temperature > 650 {
		return errors.New("temperature must be 0..650 °C")
	}
	if v.Mode != "constant" && v.Mode != "sine" {
		return errors.New("mode must be constant or sine")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = v
	return nil
}
func (s *simulator) recordPublish(protocol string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.publishers[protocol] = "unavailable"
	} else {
		s.publishers[protocol] = "publishing"
	}
}
func (s *simulator) publisherStatus() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := map[string]string{}
	for k, v := range s.publishers {
		result[k] = v
	}
	return result
}

func startOPCUA(ctx context.Context, sim *simulator, host string, port int) (*server.Server, error) {
	srv := server.New(server.EndPoint(host, port), server.EnableSecurity("None", ua.MessageSecurityModeNone), server.EnableAuthMode(ua.UserTokenTypeAnonymous))
	ns := server.NewNodeNameSpace(srv, "OT.io Presense")
	root, _ := srv.Namespace(0)
	root.Objects().AddRef(ns.Objects(), id.HasComponent, true)
	for _, name := range []string{"Temperature", "Running"} {
		node := ns.AddNewVariableStringNode(name, func() *ua.DataValue {
			v := sim.snapshot()
			if name == "Running" {
				return server.DataValueFromValue(v.Running)
			}
			return server.DataValueFromValue(v.Temperature)
		})
		ns.Objects().AddRef(node, id.HasComponent, true)
	}
	if err := srv.Start(ctx); err != nil {
		return nil, err
	}
	return srv, nil
}

func crc16(data []byte) uint16 {
	crc := uint16(0xffff)
	for _, b := range data {
		crc ^= uint16(b)
		for range 8 {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xa001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func serveModbus(ctx context.Context, l net.Listener, sim *simulator, rtu bool) {
	stop := context.AfterFunc(ctx, func() { _ = l.Close() })
	defer stop()
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
			defer stop()
			for {
				_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
				var unit byte
				var pdu, header []byte
				if rtu {
					request := make([]byte, 8)
					if _, err := io.ReadFull(conn, request); err != nil {
						return
					}
					if binary.LittleEndian.Uint16(request[6:]) != crc16(request[:6]) {
						return
					}
					unit = request[0]
					pdu = request[1:6]
				} else {
					header = make([]byte, 7)
					if _, err := io.ReadFull(conn, header); err != nil {
						return
					}
					size := int(binary.BigEndian.Uint16(header[4:6]))
					if binary.BigEndian.Uint16(header[2:4]) != 0 || size < 2 || size > 254 {
						return
					}
					unit = header[6]
					pdu = make([]byte, size-1)
					if _, err := io.ReadFull(conn, pdu); err != nil {
						return
					}
				}
				response := modbusResponse(sim, unit, pdu)
				if rtu {
					response = append([]byte{unit}, response...)
					crc := crc16(response)
					response = append(response, byte(crc), byte(crc>>8))
				} else {
					binary.BigEndian.PutUint16(header[4:6], uint16(len(response)+1))
					response = append(header, response...)
				}
				if _, err := conn.Write(response); err != nil {
					return
				}
			}
		}()
	}
}
func modbusResponse(sim *simulator, unit byte, pdu []byte) []byte {
	if len(pdu) == 0 {
		return []byte{0x80, 3}
	}
	if unit != 1 {
		return []byte{pdu[0] | 0x80, 11}
	}
	if pdu[0] != 3 {
		return []byte{pdu[0] | 0x80, 1}
	}
	if len(pdu) != 5 {
		return []byte{0x83, 3}
	}
	start, count := int(binary.BigEndian.Uint16(pdu[1:3])), int(binary.BigEndian.Uint16(pdu[3:5]))
	if count < 1 || count > 125 {
		return []byte{0x83, 3}
	}
	if start+count > 2 {
		return []byte{0x83, 2}
	}
	v := sim.snapshot()
	registers := []uint16{uint16(math.Round(v.Temperature * 100)), 0}
	if v.Running {
		registers[1] = 1
	}
	response := make([]byte, 2+count*2)
	response[0], response[1] = 3, byte(count*2)
	for i := range count {
		binary.BigEndian.PutUint16(response[2+i*2:], registers[start+i])
	}
	return response
}

func listenModbus(ctx context.Context, sim *simulator, port int, rtu bool) (net.Listener, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err == nil {
		go serveModbus(ctx, l, sim, rtu)
	}
	return l, err
}
