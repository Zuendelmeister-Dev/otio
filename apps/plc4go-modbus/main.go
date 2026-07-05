package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type registerBank struct {
	mu        sync.RWMutex
	registers [100]uint16
}

func envString(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func envFloat(name string, fallback float64) float64 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func scaled(value float64, scale float64) uint16 {
	if value < 0 {
		return 0
	}
	return uint16(math.Round(value / scale))
}

func (bank *registerBank) set(values []uint16) {
	bank.mu.Lock()
	defer bank.mu.Unlock()
	copy(bank.registers[:], values)
}

func (bank *registerBank) read(address uint16, count uint16) ([]uint16, bool) {
	bank.mu.RLock()
	defer bank.mu.RUnlock()
	end := int(address) + int(count)
	if count == 0 || end > len(bank.registers) {
		return nil, false
	}
	values := make([]uint16, count)
	copy(values, bank.registers[address:uint16(end)])
	return values, true
}

func generatorValue(mode string, elapsed float64, base float64, index int) float64 {
	switch mode {
	case "sawtooth":
		return base + math.Mod(elapsed*float64(index+1), 60)
	case "triangle":
		period := 30.0
		phase := math.Mod(elapsed, period) / period
		if phase < 0.5 {
			return base + phase*20
		}
		return base + (1-phase)*20
	default:
		return base + math.Sin(elapsed/5+float64(index))*2
	}
}

func startGenerator(bank *registerBank, mode string, currentBase float64, rpmBase float64, temperatureBase float64, vibrationBase float64) {
	start := time.Now()
	for {
		elapsed := time.Since(start).Seconds()
		current := generatorValue(mode, elapsed, currentBase, 0)
		rpm := generatorValue(mode, elapsed, rpmBase, 1)
		temperature := generatorValue(mode, elapsed, temperatureBase, 2)
		vibration := generatorValue(mode, elapsed, vibrationBase, 3)
		bank.set([]uint16{
			scaled(current, 0.01),
			scaled(rpm, 1),
			scaled(temperature, 0.1),
			scaled(vibration, 0.001),
		})
		time.Sleep(500 * time.Millisecond)
	}
}

func handleModbusConnection(conn net.Conn, bank *registerBank) {
	defer conn.Close()
	for {
		header := make([]byte, 7)
		if _, err := io.ReadFull(conn, header); err != nil {
			return
		}
		length := binary.BigEndian.Uint16(header[4:6])
		if length == 0 || length > 260 {
			return
		}
		pdu := make([]byte, int(length)-1)
		if _, err := io.ReadFull(conn, pdu); err != nil {
			return
		}
		if len(pdu) < 5 || pdu[0] != 3 {
			return
		}
		address := binary.BigEndian.Uint16(pdu[1:3])
		count := binary.BigEndian.Uint16(pdu[3:5])
		values, ok := bank.read(address, count)
		if !ok {
			return
		}
		responsePDU := make([]byte, 2+len(values)*2)
		responsePDU[0] = 3
		responsePDU[1] = byte(len(values) * 2)
		for i, value := range values {
			binary.BigEndian.PutUint16(responsePDU[2+i*2:4+i*2], value)
		}
		response := make([]byte, 7+len(responsePDU))
		copy(response[0:4], header[0:4])
		binary.BigEndian.PutUint16(response[4:6], uint16(len(responsePDU)+1))
		response[6] = header[6]
		copy(response[7:], responsePDU)
		if _, err := conn.Write(response); err != nil {
			return
		}
	}
}

func startModbusServer(address string, bank *registerBank) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("PLC4Go-style Modbus endpoint listening on", address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleModbusConnection(conn, bank)
	}
}

func main() {
	deviceID := envString("DEVICE_ID", "plc4go-modbus-01")
	mode := envString("GENERATOR_MODE", "triangle")
	modbusPort := envInt("MODBUS_PORT", 5020)
	uiPort := envInt("PLC4GO_UI_PORT", 8400)
	currentBase := envFloat("CURRENT_BASE", 8.0)
	rpmBase := envFloat("RPM_BASE", 1400.0)
	temperatureBase := envFloat("TEMP_BASE", 25.0)
	vibrationBase := envFloat("VIBRATION_BASE", 0.7)

	bank := &registerBank{}
	go startGenerator(bank, mode, currentBase, rpmBase, temperatureBase, vibrationBase)
	go startModbusServer(fmt.Sprintf(":%d", modbusPort), bank)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "deviceId": deviceID, "adapter": "plc4go", "protocol": "modbus-tcp"})
	})
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"deviceId": deviceID, "adapter": "plc4go", "protocol": "modbus-tcp", "port": modbusPort, "uiPort": uiPort, "generatorMode": mode})
	})
	log.Println("PLC4Go-style machine UI listening on", uiPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", uiPort), nil))
}
