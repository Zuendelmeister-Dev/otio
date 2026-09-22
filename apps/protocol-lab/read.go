package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	plc "github.com/apache/plc4x/plc4go/pkg/api"
	"github.com/apache/plc4x/plc4go/pkg/api/config"
	"github.com/apache/plc4x/plc4go/pkg/api/drivers"
	"github.com/apache/plc4x/plc4go/pkg/api/model"
	"github.com/apache/plc4x/plc4go/pkg/api/values"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/rs/zerolog"
)

type ReadRequest struct {
	Protocol   string `json:"protocol"`
	Connection string `json:"connection"`
	Address    string `json:"address"`
}

func validateRead(r ReadRequest) error {
	c, ok := capability(r.Protocol)
	if !ok || c.Connection == "" {
		return errors.New("protocol requires a hardware gateway or is not implemented")
	}
	if len(r.Connection) > 2048 || len(r.Address) > 1024 || strings.TrimSpace(r.Address) == "" {
		return errors.New("a non-empty address and a bounded connection URL are required")
	}
	schemes := map[string]string{"modbus-tcp": "modbus-tcp", "modbus-rtu-tcp": "modbus-rtu:tcp", "opcua-tcp": "opc.tcp", "s7": "s7", "ethernet-ip": "eip", "bacnet-ip": "bacnet-ip", "knxnet-ip": "knxnet-ip", "iec-60870-5-104": "iec-60870-5-104", "mqtt": "tcp", "amqp": "amqp"}
	prefix := schemes[r.Protocol] + "://"
	if !strings.HasPrefix(r.Connection, prefix) {
		return fmt.Errorf("connection must start with %s", prefix)
	}
	// Normalize the PLC4X compound scheme for URL validation only.
	u, err := url.Parse(strings.Replace(r.Connection, prefix, "tcp://", 1))
	if err != nil || u.Hostname() == "" {
		return errors.New("invalid connection host")
	}
	return nil
}

func readValue(ctx context.Context, r ReadRequest) (any, error) {
	if err := validateRead(r); err != nil {
		return nil, err
	}
	switch r.Protocol {
	case "mqtt":
		return sampleMQTT(ctx, r)
	case "amqp":
		return sampleAMQP(ctx, r)
	case "opcua-tcp":
		node, err := ua.ParseNodeID(r.Address)
		if err != nil {
			return nil, err
		}
		client, err := opcua.NewClient(r.Connection, opcua.SecurityMode(ua.MessageSecurityModeNone), opcua.SecurityPolicy(ua.SecurityPolicyURINone), opcua.AutoReconnect(false))
		if err != nil {
			return nil, err
		}
		if err = client.Connect(ctx); err != nil {
			return nil, err
		}
		defer client.Close(ctx)
		response, err := client.Read(ctx, &ua.ReadRequest{NodesToRead: []*ua.ReadValueID{{NodeID: node, AttributeID: ua.AttributeIDValue}}})
		if err != nil {
			return nil, err
		}
		if len(response.Results) != 1 || response.Results[0].Status != ua.StatusOK {
			return nil, errors.New("OPC UA node returned a bad status")
		}
		if response.Results[0].Value == nil {
			return nil, errors.New("OPC UA node has no value")
		}
		return response.Results[0].Value.Value(), nil
	}
	quiet := config.WithCustomLogger(zerolog.Nop())
	manager := plc.NewPlcDriverManager(quiet)
	defer manager.Close()
	switch r.Protocol {
	case "modbus-tcp":
		drivers.RegisterModbusTcpDriver(manager, quiet)
	case "modbus-rtu-tcp":
		drivers.RegisterModbusRtuDriver(manager, quiet)
	case "s7":
		drivers.RegisterS7Driver(manager, quiet)
	case "ethernet-ip":
		drivers.RegisterEipDriver(manager, quiet)
	case "bacnet-ip":
		drivers.RegisterBacnetDriver(manager, quiet)
	case "knxnet-ip":
		drivers.RegisterKnxDriver(manager, quiet)
	case "iec-60870-5-104":
		drivers.RegisterIec608705104Driver(manager, quiet)
	}
	conn, err := manager.GetConnection(ctx, r.Connection)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if conn.GetMetadata().CanRead() {
		request, err := conn.ReadRequestBuilder().AddTagAddress("value", r.Address).Build()
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result, ok := <-request.Execute(ctx):
			if !ok || result == nil {
				return nil, errors.New("driver closed without a result")
			}
			if result.GetErr() != nil {
				return nil, result.GetErr()
			}
			response := result.GetResponse()
			if response == nil || response.GetResponseCode("value") != model.PlcResponseCode_OK {
				return nil, errors.New("driver returned a bad response status")
			}
			return plainValue(response.GetValue("value")), nil
		}
	}
	if !conn.GetMetadata().CanSubscribe() {
		return nil, errors.New("driver does not support reading or event sampling")
	}
	events := make(chan any, 1)
	request, err := conn.SubscriptionRequestBuilder().AddChangeOfStateTagAddress("value", r.Address).AddPreRegisteredConsumer("value", func(event model.PlcSubscriptionEvent) {
		if event.GetResponseCode("value") == model.PlcResponseCode_OK {
			select {
			case events <- plainValue(event.GetValue("value")):
			default:
			}
		}
	}).Build()
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-request.Execute(ctx):
		if result == nil {
			return nil, errors.New("empty subscription result")
		}
		if result.GetErr() != nil {
			return nil, result.GetErr()
		}
		if result.GetResponse() == nil || result.GetResponse().GetResponseCode("value") != model.PlcResponseCode_OK {
			return nil, errors.New("driver rejected the subscription")
		}
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case value := <-events:
		return value, nil
	}
}

func plainValue(v values.PlcValue) any {
	if v == nil || v.IsNull() {
		return nil
	}
	if v.IsList() {
		result := []any{}
		for _, item := range v.GetList() {
			result = append(result, plainValue(item))
		}
		return result
	}
	if v.IsStruct() {
		result := map[string]any{}
		for key, item := range v.GetStruct() {
			result[key] = plainValue(item)
		}
		return result
	}
	switch v.GetPlcValueType() {
	case values.BOOL:
		return v.GetBool()
	case values.SINT, values.INT, values.DINT, values.LINT:
		return v.GetInt64()
	case values.BYTE, values.WORD, values.DWORD, values.LWORD, values.USINT, values.UINT, values.UDINT, values.ULINT:
		return v.GetUint64()
	case values.REAL, values.LREAL:
		return v.GetFloat64()
	case values.STRING, values.WSTRING, values.CHAR, values.WCHAR:
		return v.GetString()
	}
	return v.String()
}

func jsonValue(raw []byte) (any, error) {
	if len(raw) > 1024*1024 {
		return nil, errors.New("message exceeds 1 MiB")
	}
	var value any
	err := json.Unmarshal(raw, &value)
	return value, err
}

const readTimeout = 8 * time.Second
