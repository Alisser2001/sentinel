package config

type SentinelConfig struct {
	CPUThreshold  float64            `json:"cpu_threshold"`
	MemThreshold  float64            `json:"mem_threshold"`
	ActiveWebhook string             `json:"active_webhook"`
	Webhooks      map[string]string  `json:"webhooks"`
}