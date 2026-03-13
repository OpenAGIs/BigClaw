package domain

import "fmt"

var allowedTransitions = map[TaskState]map[TaskState]struct{}{
	TaskQueued: {
		TaskLeased:    {},
		TaskCancelled: {},
	},
	TaskLeased: {
		TaskRunning:   {},
		TaskRetrying:  {},
		TaskCancelled: {},
	},
	TaskRunning: {
		TaskSucceeded:  {},
		TaskFailed:     {},
		TaskBlocked:    {},
		TaskRetrying:   {},
		TaskCancelled:  {},
		TaskDeadLetter: {},
	},
	TaskBlocked: {
		TaskRetrying:  {},
		TaskCancelled: {},
	},
	TaskRetrying: {
		TaskQueued:     {},
		TaskDeadLetter: {},
		TaskCancelled:  {},
	},
	TaskFailed: {
		TaskRetrying:   {},
		TaskDeadLetter: {},
	},
}

func CanTransition(from, to TaskState) bool {
	nextStates, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	_, ok = nextStates[to]
	return ok
}

func ValidateTransition(from, to TaskState) error {
	if from == to {
		return nil
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid task transition: %s -> %s", from, to)
	}
	return nil
}
