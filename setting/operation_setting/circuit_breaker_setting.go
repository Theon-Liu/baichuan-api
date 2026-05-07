package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// CircuitBreakerSetting 渠道熔断配置
type CircuitBreakerSetting struct {
	// Enabled 是否启用熔断机制
	Enabled bool `json:"enabled"`
	// WindowSize 滑动窗口大小（请求次数）
	WindowSize int `json:"window_size"`
	// FailureThreshold 窗口内失败次数阈值，达到后触发熔断
	FailureThreshold int `json:"failure_threshold"`
	// InitialCooldownSeconds 首次熔断冷却时间（秒）
	InitialCooldownSeconds int `json:"initial_cooldown_seconds"`
	// MaxCooldownSeconds 冷却时间上限（秒）
	MaxCooldownSeconds int `json:"max_cooldown_seconds"`
}

var circuitBreakerSetting = CircuitBreakerSetting{
	Enabled:                false,
	WindowSize:             10,
	FailureThreshold:       3,
	InitialCooldownSeconds: 300,  // 5分钟
	MaxCooldownSeconds:     7200, // 2小时
}

func init() {
	config.GlobalConfig.Register("circuit_breaker_setting", &circuitBreakerSetting)
}

func GetCircuitBreakerSetting() *CircuitBreakerSetting {
	return &circuitBreakerSetting
}
