package worker

import "time"

type BackoffReason string

const (
	BackoffTakeoverHold   BackoffReason = "takeover_hold"
	BackoffSchedulerRetry BackoffReason = "scheduler_retry"
	BackoffPreemption     BackoffReason = "preemption_retry"
	BackoffExecutionRetry BackoffReason = "execution_retry"
)

type BackoffRequest struct {
	Attempt   int
	Reason    BackoffReason
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

type BackoffStrategy interface {
	Name() string
	Delay(BackoffRequest) time.Duration
}

type FixedBackoffStrategy struct{}

func (FixedBackoffStrategy) Name() string { return "fixed" }

func (FixedBackoffStrategy) Delay(request BackoffRequest) time.Duration {
	return clampBackoffDelay(request.BaseDelay, request.MaxDelay)
}

type LinearBackoffStrategy struct{}

func (LinearBackoffStrategy) Name() string { return "linear" }

func (LinearBackoffStrategy) Delay(request BackoffRequest) time.Duration {
	attempt := normalizedAttempt(request.Attempt)
	return clampBackoffDelay(time.Duration(attempt)*request.BaseDelay, request.MaxDelay)
}

type ExponentialBackoffStrategy struct{}

func (ExponentialBackoffStrategy) Name() string { return "exponential" }

func (ExponentialBackoffStrategy) Delay(request BackoffRequest) time.Duration {
	attempt := normalizedAttempt(request.Attempt)
	multiplier := 1 << (attempt - 1)
	return clampBackoffDelay(time.Duration(multiplier)*request.BaseDelay, request.MaxDelay)
}

type BackoffPolicy struct {
	Strategy             BackoffStrategy
	TakeoverHoldDelay    time.Duration
	SchedulerRetryDelay  time.Duration
	PreemptionRetryDelay time.Duration
	ExecutionRetryDelay  time.Duration
	MaxDelay             time.Duration
}

func DefaultBackoffPolicy() BackoffPolicy {
	return BackoffPolicy{
		Strategy:             FixedBackoffStrategy{},
		TakeoverHoldDelay:    250 * time.Millisecond,
		SchedulerRetryDelay:  100 * time.Millisecond,
		PreemptionRetryDelay: 100 * time.Millisecond,
		ExecutionRetryDelay:  200 * time.Millisecond,
		MaxDelay:             5 * time.Second,
	}
}

func (p BackoffPolicy) Resolve(reason BackoffReason, attempt int) time.Duration {
	p = p.withDefaults()
	request := BackoffRequest{
		Attempt:   attempt,
		Reason:    reason,
		BaseDelay: p.baseDelay(reason),
		MaxDelay:  p.MaxDelay,
	}
	return p.Strategy.Delay(request)
}

func (p BackoffPolicy) withDefaults() BackoffPolicy {
	defaults := DefaultBackoffPolicy()
	if p.Strategy == nil {
		p.Strategy = defaults.Strategy
	}
	if p.TakeoverHoldDelay <= 0 {
		p.TakeoverHoldDelay = defaults.TakeoverHoldDelay
	}
	if p.SchedulerRetryDelay <= 0 {
		p.SchedulerRetryDelay = defaults.SchedulerRetryDelay
	}
	if p.PreemptionRetryDelay <= 0 {
		p.PreemptionRetryDelay = defaults.PreemptionRetryDelay
	}
	if p.ExecutionRetryDelay <= 0 {
		p.ExecutionRetryDelay = defaults.ExecutionRetryDelay
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = defaults.MaxDelay
	}
	return p
}

func (p BackoffPolicy) baseDelay(reason BackoffReason) time.Duration {
	switch reason {
	case BackoffTakeoverHold:
		return p.TakeoverHoldDelay
	case BackoffSchedulerRetry:
		return p.SchedulerRetryDelay
	case BackoffPreemption:
		return p.PreemptionRetryDelay
	case BackoffExecutionRetry:
		return p.ExecutionRetryDelay
	default:
		return p.ExecutionRetryDelay
	}
}

func normalizedAttempt(attempt int) int {
	if attempt < 1 {
		return 1
	}
	return attempt
}

func clampBackoffDelay(delay, maxDelay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	if maxDelay > 0 && delay > maxDelay {
		return maxDelay
	}
	return delay
}
