package main

// Capability describes implemented behavior, not just the availability of an IP port.
type Capability struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Transport  string `json:"transport"`
	Status     string `json:"status"`
	Notes      string `json:"notes"`
	Connection string `json:"connection"`
	Address    string `json:"address"`
	Simulator  bool   `json:"simulator"`
}

var catalog = []Capability{
	{"modbus-tcp", "Modbus TCP", "TCP", "read", "Holding registers. Local Presense simulator included.", "modbus-tcp://localhost:1502?default-unit-identifier=1", "holding-register:1:UINT", true},
	{"modbus-rtu-tcp", "Modbus RTU over TCP", "TCP gateway", "read", "RTU framing with CRC over a transparent TCP tunnel. Native serial RTU requires a gateway.", "modbus-rtu:tcp://localhost:1503?default-unit-identifier=1", "holding-register:1:UINT", true},
	{"opcua-tcp", "OPC UA Binary", "TCP", "read", "Native OPC UA read, separate from the existing HTTP demo. Lab uses anonymous / SecurityPolicy None.", "opc.tcp://localhost:4842", "ns=1;s=Temperature", true},
	{"s7", "Siemens S7 / RFC1006", "TCP", "experimental", "PLC4Go S7 read. Requires a reachable PLC with permitted S7 access; does not implement PROFINET IO.", "s7://192.168.1.10:102?remote-rack=0&remote-slot=1", "%DB1:0:REAL", false},
	{"ethernet-ip", "EtherNet/IP (CIP)", "TCP", "experimental", "PLC4Go explicit CIP symbolic tag reads. Not cyclic I/O or a certified adapter.", "eip://192.168.1.10:44818", "%Temperature:REAL", false},
	{"bacnet-ip", "BACnet/IP", "UDP", "experimental", "PLC4Go ReadProperty. Requires an IP device, not an MS/TP serial network.", "bacnet-ip://192.168.1.10:47808", "0,1/85", false},
	{"knxnet-ip", "KNXnet/IP", "UDP", "experimental", "PLC4Go group communication; read where supported, otherwise wait for one group event. A tunnelling gateway and matching DPT are required.", "knxnet-ip://192.168.1.10:3671", "1/2/3:DPT_Value_Temp", false},
	{"iec-60870-5-104", "IEC 60870-5-104", "TCP", "experimental", "PLC4Go event sample: wait for the first matching ASDU/IOA update. A timeout does not mean a point is zero.", "iec-60870-5-104://192.168.1.10:2404", "1/1", false},
	{"mqtt", "MQTT 3.1.1", "TCP", "sample", "Sample the next or retained JSON message on a topic. Local retained demo publisher included.", "tcp://localhost:1883", "lab/temperature", true},
	{"amqp", "AMQP 0-9-1", "TCP", "sample", "Sample an exchange routing key via an exclusive temporary queue. Does not consume another application's queue. AMQP 1.0 is not implemented.", "amqp://guest:guest@localhost:5672/", "amq.topic/lab.temperature", true},
	{"profinet", "PROFINET", "Ethernet / hardware", "gateway", "Cyclic RT/IRT uses Ethernet frames and requires a controller stack. Use an OPC UA, S7 or MQTT gateway; no native PROFINET IO driver.", "", "", false},
	{"profibus", "PROFIBUS DP / PA", "Serial fieldbus", "gateway", "Requires a PROFIBUS master and hardware gateway.", "", "", false},
	{"ethercat", "EtherCAT", "Raw Ethernet", "gateway", "Requires an EtherCAT master, dedicated interface and suitable timing. TCP transport is not EtherCAT process I/O.", "", "", false},
	{"canopen", "CAN / CANopen", "CAN bus", "gateway", "Requires a CAN interface or a gateway with a documented application mapping.", "", "", false},
	{"j1939", "J1939", "CAN bus", "gateway", "Requires a CAN/J1939 gateway and PGN/SPN mapping.", "", "", false},
	{"io-link", "IO-Link", "Point-to-point", "gateway", "Use an IO-Link master exposing OPC UA, MQTT or another implemented IP protocol. No universal TCP device protocol.", "", "", false},
	{"hart", "HART / HART-IP", "Fieldbus / IP variant", "unimplemented", "Wired HART needs a modem/gateway. HART-IP is network based but a HART-IP command adapter is not implemented.", "", "", false},
	{"mbus", "M-Bus / Wireless M-Bus", "Wired / radio", "gateway", "Requires a level converter or radio receiver and a gateway-specific mapping. TCP tunnelling alone does not decode meter telegrams.", "", "", false},
	{"iec61850", "IEC 61850", "MMS TCP / Ethernet", "unimplemented", "MMS is IP based but not implemented. GOOSE and sampled values require separate Ethernet support; an OPC UA/MQTT gateway is an alternative.", "", "", false},
	{"dnp3", "DNP3", "TCP / serial", "unimplemented", "DNP3 over TCP is applicable but no native outstation/master adapter is integrated yet. Use a gateway; no simulated support claim.", "", "", false},
	{"asi", "AS-Interface (AS-i)", "Fieldbus", "gateway", "Requires an AS-i master with an implemented upstream IP protocol.", "", "", false},
}

func capability(id string) (Capability, bool) {
	for _, c := range catalog {
		if c.ID == id {
			return c, true
		}
	}
	return Capability{}, false
}
