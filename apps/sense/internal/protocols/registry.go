package protocols

import "fmt"

type Registry struct {
	readers     map[string]Reader
	subscribers map[string]Subscriber
}

func NewRegistry() Registry {
	opcua := OPCUAHTTPReader{}
	registry := Registry{
		readers:     map[string]Reader{},
		subscribers: map[string]Subscriber{},
	}
	registry.RegisterReader("modbus-tcp", ModbusTCPReader{})
	registry.RegisterReader("opcua", opcua)
	registry.RegisterSubscriber("opcua", opcua)
	return registry
}

func (registry Registry) RegisterReader(sourceType string, reader Reader) {
	registry.readers[sourceType] = reader
}

func (registry Registry) RegisterSubscriber(sourceType string, subscriber Subscriber) {
	registry.subscribers[sourceType] = subscriber
}

func (registry Registry) Reader(sourceType string) (Reader, error) {
	reader, ok := registry.readers[sourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported source type %q", sourceType)
	}
	return reader, nil
}

func (registry Registry) Subscriber(sourceType string) (Subscriber, error) {
	subscriber, ok := registry.subscribers[sourceType]
	if !ok {
		return nil, fmt.Errorf("source type %q does not support subscriptions", sourceType)
	}
	return subscriber, nil
}

func (registry Registry) SupportedSourceTypes() []string {
	types := make([]string, 0, len(registry.readers))
	for sourceType := range registry.readers {
		types = append(types, sourceType)
	}
	return types
}
