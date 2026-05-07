package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// channelCircuitState 单个渠道的熔断状态（纯内存，重启后重置）
type channelCircuitState struct {
	mu sync.Mutex

	// 滑动窗口：环形缓冲区，记录最近 WindowSize 次请求的结果（true=失败）
	window    []bool
	windowPos int // 下一次写入的位置
	filled    bool // 窗口是否已填满（即请求次数 >= WindowSize）

	// 熔断状态
	isOpen        bool
	cooldownUntil time.Time

	// 累计熔断次数，用于计算倍增冷却时间
	tripCount int
}

// failureCount 返回当前窗口内的失败次数
func (s *channelCircuitState) failureCount() int {
	count := 0
	size := len(s.window)
	limit := size
	if !s.filled {
		limit = s.windowPos
	}
	for i := 0; i < limit; i++ {
		if s.window[i] {
			count++
		}
	}
	return count
}

// record 记录一次请求结果，返回是否应该触发熔断
func (s *channelCircuitState) record(failed bool, setting *operation_setting.CircuitBreakerSetting) (shouldTrip bool) {
	// 写入窗口
	s.window[s.windowPos] = failed
	s.windowPos = (s.windowPos + 1) % len(s.window)
	if !s.filled && s.windowPos == 0 {
		s.filled = true
	}

	if !failed {
		return false
	}

	// 检查窗口内失败次数是否达到阈值
	if s.failureCount() >= setting.FailureThreshold {
		s.trip(setting)
		return true
	}
	return false
}

// trip 触发熔断，计算冷却时间（首次基础值，之后每次翻倍）
func (s *channelCircuitState) trip(setting *operation_setting.CircuitBreakerSetting) {
	s.tripCount++

	// 冷却时间 = InitialCooldown * 2^(tripCount-1)，上限 MaxCooldown
	cooldown := setting.InitialCooldownSeconds
	for i := 1; i < s.tripCount; i++ {
		cooldown *= 2
		if cooldown >= setting.MaxCooldownSeconds {
			cooldown = setting.MaxCooldownSeconds
			break
		}
	}

	s.isOpen = true
	s.cooldownUntil = time.Now().Add(time.Duration(cooldown) * time.Second)

	// 熔断后清空窗口，避免冷却结束后立刻重新触发
	for i := range s.window {
		s.window[i] = false
	}
	s.windowPos = 0
	s.filled = false
}

// isAvailable 判断渠道当前是否可用
// 返回 false 表示正在熔断
func (s *channelCircuitState) isAvailable() bool {
	if !s.isOpen {
		return true
	}
	if time.Now().After(s.cooldownUntil) {
		// 冷却期结束，自动恢复
		s.isOpen = false
		return true
	}
	return false
}

// cooldownRemaining 返回剩余冷却时间（秒）
func (s *channelCircuitState) cooldownRemaining() int {
	if !s.isOpen {
		return 0
	}
	remaining := time.Until(s.cooldownUntil).Seconds()
	if remaining < 0 {
		return 0
	}
	return int(remaining)
}

// ------- 全局注册表 -------

var (
	circuitBreakerMu    sync.RWMutex
	circuitBreakerStates = make(map[int]*channelCircuitState)
)

func getOrCreateCircuitState(channelID int, windowSize int) *channelCircuitState {
	circuitBreakerMu.RLock()
	state, ok := circuitBreakerStates[channelID]
	circuitBreakerMu.RUnlock()
	if ok {
		return state
	}

	circuitBreakerMu.Lock()
	defer circuitBreakerMu.Unlock()
	// double-check
	if state, ok = circuitBreakerStates[channelID]; ok {
		return state
	}
	state = &channelCircuitState{
		window: make([]bool, windowSize),
	}
	circuitBreakerStates[channelID] = state
	return state
}

// IsChannelAvailable 判断渠道是否可用（未熔断）
func IsChannelAvailable(channelID int) bool {
	setting := operation_setting.GetCircuitBreakerSetting()
	if !setting.Enabled {
		return true
	}

	circuitBreakerMu.RLock()
	state, ok := circuitBreakerStates[channelID]
	circuitBreakerMu.RUnlock()
	if !ok {
		return true
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	return state.isAvailable()
}

// RecordChannelResult 记录一次请求结果，返回是否刚刚触发熔断及当前冷却秒数
func RecordChannelResult(channelID int, failed bool) (tripped bool, cooldownSeconds int) {
	setting := operation_setting.GetCircuitBreakerSetting()
	if !setting.Enabled {
		return false, 0
	}

	state := getOrCreateCircuitState(channelID, setting.WindowSize)
	state.mu.Lock()
	defer state.mu.Unlock()

	tripped = state.record(failed, setting)
	if tripped {
		cooldownSeconds = state.cooldownRemaining()
	}
	return
}

// GetCircuitBreakerInfo 获取渠道熔断信息（用于日志/监控）
func GetCircuitBreakerInfo(channelID int) (isOpen bool, cooldownSeconds int, tripCount int) {
	circuitBreakerMu.RLock()
	state, ok := circuitBreakerStates[channelID]
	circuitBreakerMu.RUnlock()
	if !ok {
		return false, 0, 0
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	_ = state.isAvailable() // 触发自动恢复检查
	return state.isOpen, state.cooldownRemaining(), state.tripCount
}

// GetAllCircuitBreakerStatus 获取所有渠道的熔断状态（用于管理后台）
type ChannelCircuitStatus struct {
	ChannelID       int  `json:"channel_id"`
	IsOpen          bool `json:"is_open"`
	CooldownSeconds int  `json:"cooldown_seconds"`
	TripCount       int  `json:"trip_count"`
}

func GetAllCircuitBreakerStatus() []ChannelCircuitStatus {
	circuitBreakerMu.RLock()
	ids := make([]int, 0, len(circuitBreakerStates))
	for id := range circuitBreakerStates {
		ids = append(ids, id)
	}
	circuitBreakerMu.RUnlock()

	result := make([]ChannelCircuitStatus, 0, len(ids))
	for _, id := range ids {
		isOpen, cooldown, trips := GetCircuitBreakerInfo(id)
		if isOpen || trips > 0 {
			result = append(result, ChannelCircuitStatus{
				ChannelID:       id,
				IsOpen:          isOpen,
				CooldownSeconds: cooldown,
				TripCount:       trips,
			})
		}
	}
	return result
}

// ResetCircuitBreaker 手动重置某个渠道的熔断状态
func ResetCircuitBreaker(channelID int) {
	circuitBreakerMu.Lock()
	delete(circuitBreakerStates, channelID)
	circuitBreakerMu.Unlock()
	common.SysLog(fmt.Sprintf("渠道 #%d 熔断状态已手动重置", channelID))
}
