package llm

import (
	"fmt"
	"sync"
	"time"
)

// CircuitState 熔断器状态
type CircuitState int

const (
	// StateClosed 关闭状态（正常放行请求）
	StateClosed CircuitState = iota
	// StateOpen 打开状态（拒绝所有请求）
	StateOpen
	// StateHalfOpen 半开状态（允许有限探测请求）
	StateHalfOpen
)

// String 返回状态的可读名称
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker 三态熔断器。
//
// 状态转换：
//
//	CLOSED --（连续失败达阈值）--> OPEN
//	OPEN   --（超时后自动）------> HALF_OPEN
//	HALF_OPEN --（探测成功达阈值）--> CLOSED
//	HALF_OPEN --（探测失败）-------> OPEN
type CircuitBreaker struct {
	mu sync.RWMutex

	state            CircuitState
	failureCount     int       // 当前连续失败次数
	successCount     int       // 半开状态下的连续成功次数
	lastFailureTime  time.Time // 最后一次失败时间

	failureThreshold int           // 触发熔断的连续失败次数
	successThreshold int           // 半开恢复所需的连续成功次数
	timeout          time.Duration // 从 OPEN 到 HALF_OPEN 的冷却时间

	onStateChange func(from, to CircuitState) // 状态变更回调
}

// NewCircuitBreaker 创建一个新的三态熔断器
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// SetOnStateChange 设置状态变更回调函数
func (cb *CircuitBreaker) SetOnStateChange(fn func(from, to CircuitState)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// Allow 判断当前是否允许请求通过。
// 返回 true 表示允许，返回 false 表示熔断器打开，应拒绝请求。
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// 检查冷却时间是否已过，自动转入半开状态
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false

	case StateHalfOpen:
		// 半开状态允许请求通过以进行探测
		return true

	default:
		return false
	}
}

// RecordSuccess 记录一次成功的请求。
// 在半开状态下，连续成功达到阈值后将恢复到关闭状态。
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failureCount = 0

	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.transitionTo(StateClosed)
		}
	}
}

// RecordFailure 记录一次失败的请求。
// 在关闭状态下连续失败达到阈值将触发熔断；在半开状态下任何失败立即重新打开。
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.transitionTo(StateOpen)
		}

	case StateHalfOpen:
		// 半开状态下失败，立即重新打开熔断器
		cb.transitionTo(StateOpen)
	}
}

// State 返回熔断器的当前状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats 返回熔断器的统计信息
func (cb *CircuitBreaker) Stats() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return fmt.Sprintf("state=%s failures=%d successes=%d",
		cb.state, cb.failureCount, cb.successCount)
}

// transitionTo 执行状态转换（调用前必须持有锁）
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	// 重置计数器
	switch newState {
	case StateClosed:
		cb.failureCount = 0
		cb.successCount = 0
	case StateOpen:
		cb.successCount = 0
	case StateHalfOpen:
		cb.successCount = 0
		cb.failureCount = 0
	}

	if cb.onStateChange != nil {
		// 在新 goroutine 中执行回调，避免阻塞
		go cb.onStateChange(oldState, newState)
	}
}
