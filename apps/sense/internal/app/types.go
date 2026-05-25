package app

type Config struct {
	Broker               BrokerConfig   `json:"broker"`
	PollIntervalMS       int            `json:"pollIntervalMs"`
	HealthTimeoutSeconds int            `json:"healthTimeoutSeconds"`
	Sources              []SourceConfig `json:"sources"`
}

type BrokerConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	ClientID    string `json:"clientId"`
	TopicPrefix string `json:"topicPrefix"`
}

type SourceConfig struct {
	AgentID                string         `json:"agentId"`
	Type                   string         `json:"type"`
	Origin                 string         `json:"origin,omitempty"`
	DisplayName            string         `json:"displayName,omitempty"`
	Host                   string         `json:"host"`
	Port                   int            `json:"port"`
	UnitID                 byte           `json:"unitId"`
	ReadMode               string         `json:"readMode,omitempty"`
	SubscriptionIntervalMS int            `json:"subscriptionIntervalMs,omitempty"`
	Options                map[string]any `json:"options,omitempty"`
	Metrics                []MetricConfig `json:"metrics"`
}

type MetricConfig struct {
	Name     string         `json:"name"`
	Register uint16         `json:"register,omitempty"`
	NodeID   string         `json:"nodeId,omitempty"`
	Path     string         `json:"path,omitempty"`
	Address  string         `json:"address,omitempty"`
	Scale    float64        `json:"scale"`
	Unit     string         `json:"unit"`
	Type     string         `json:"type"`
	Options  map[string]any `json:"options,omitempty"`
}

type SourceStatus struct {
	Connected          bool   `json:"connected"`
	Healthy            bool   `json:"healthy"`
	LastRead           string `json:"lastRead"`
	LastReadAgoSeconds int64  `json:"lastReadAgoSeconds"`
	LastReadAgoHuman   string `json:"lastReadAgoHuman"`
	Message            string `json:"message"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Component string `json:"component"`
	Message   string `json:"message"`
}

type MetricPoint struct {
	Timestamp string  `json:"timestamp"`
	Epoch     int64   `json:"epoch"`
	Value     float64 `json:"value"`
	Unit      string  `json:"unit"`
}

type Proposal struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	ConfigRaw string `json:"configRaw"`
	Diff      string `json:"diff"`
}

type HistoryItem struct {
	FileName  string `json:"fileName"`
	CreatedAt string `json:"createdAt"`
	ConfigRaw string `json:"configRaw"`
}
