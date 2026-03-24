package queue

import "time"

// Lease mutation state machine:
//  1. acquire: queued -> leased, assign owner, increment attempt, set expiry.
//  2. renew: same owner+attempt may extend expiry while the lease is still live.
//  3. release mutations (ack/requeue/dead-letter): same owner+attempt may mutate
//     state only while the lease is still live.
//  4. expiry: once a live lease expires, the holder is fenced with ErrLeaseExpired
//     until the task is re-leased; after takeover, stale holders are fenced with
//     ErrLeaseNotOwned by the owner/attempt mismatch.
func validateLeaseMutation(current *item, lease *Lease, now time.Time) error {
	if !current.Leased || current.LeaseWorker != lease.WorkerID || current.Attempt != lease.Attempt {
		return ErrLeaseNotOwned
	}
	if !current.LeaseExpires.After(now) {
		return ErrLeaseExpired
	}
	return nil
}

func clearLease(current *item) {
	current.Leased = false
	current.LeaseWorker = ""
	current.LeaseExpires = time.Time{}
}
