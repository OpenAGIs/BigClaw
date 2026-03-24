package worker

import (
	"sort"
	"strings"
)

const (
	defaultHostProfile  = "clawhost"
	defaultCapacityPool = "default"
)

type CapacityBaseline struct {
	TotalParallelSlots     int                   `json:"total_parallel_slots"`
	ActiveParallelSlots    int                   `json:"active_parallel_slots"`
	AvailableParallelSlots int                   `json:"available_parallel_slots"`
	HostProfiles           []HostProfileCapacity `json:"host_profiles,omitempty"`
	Pools                  []PoolCapacity        `json:"pools,omitempty"`
	Nodes                  []NodeCapacity        `json:"nodes,omitempty"`
}

type HostProfileCapacity struct {
	HostProfile                string  `json:"host_profile"`
	NodeCount                  int     `json:"node_count"`
	PoolCount                  int     `json:"pool_count"`
	WorkerCount                int     `json:"worker_count"`
	ParallelSlots              int     `json:"parallel_slots"`
	ActiveParallelSlots        int     `json:"active_parallel_slots"`
	AvailableParallelSlots     int     `json:"available_parallel_slots"`
	CapacityUtilizationPercent float64 `json:"capacity_utilization_percent"`
}

type PoolCapacity struct {
	PoolID                     string  `json:"pool_id"`
	HostProfileCount           int     `json:"host_profile_count"`
	NodeCount                  int     `json:"node_count"`
	WorkerCount                int     `json:"worker_count"`
	ParallelSlots              int     `json:"parallel_slots"`
	ActiveParallelSlots        int     `json:"active_parallel_slots"`
	AvailableParallelSlots     int     `json:"available_parallel_slots"`
	CapacityUtilizationPercent float64 `json:"capacity_utilization_percent"`
}

type NodeCapacity struct {
	NodeID                     string  `json:"node_id"`
	HostProfile                string  `json:"host_profile"`
	PoolID                     string  `json:"pool_id"`
	WorkerCount                int     `json:"worker_count"`
	ParallelSlots              int     `json:"parallel_slots"`
	ActiveParallelSlots        int     `json:"active_parallel_slots"`
	AvailableParallelSlots     int     `json:"available_parallel_slots"`
	CapacityUtilizationPercent float64 `json:"capacity_utilization_percent"`
}

type hostProfileAccumulator struct {
	pools map[string]struct{}
	nodes map[string]struct{}
	HostProfileCapacity
}

type poolAccumulator struct {
	hostProfiles map[string]struct{}
	nodes        map[string]struct{}
	PoolCapacity
}

func (p *Pool) CapacityBaseline() CapacityBaseline {
	return SummarizeCapacity(p.Snapshots())
}

func SummarizeCapacity(statuses []Status) CapacityBaseline {
	baseline := CapacityBaseline{}
	if len(statuses) == 0 {
		return baseline
	}

	nodeIndex := make(map[string]*NodeCapacity)
	hostIndex := make(map[string]*hostProfileAccumulator)
	poolIndex := make(map[string]*poolAccumulator)

	for _, status := range statuses {
		nodeID := firstNonEmpty(strings.TrimSpace(status.NodeID), "unassigned")
		hostProfile := firstNonEmpty(strings.TrimSpace(status.HostProfile), defaultHostProfile)
		poolID := firstNonEmpty(strings.TrimSpace(status.PoolID), defaultCapacityPool)
		parallelSlots := status.ParallelSlots
		if parallelSlots <= 0 {
			parallelSlots = 1
		}
		activeSlots := 0
		if isActiveState(status.State) {
			activeSlots = parallelSlots
		}

		baseline.TotalParallelSlots += parallelSlots
		baseline.ActiveParallelSlots += activeSlots

		node := nodeIndex[nodeID]
		if node == nil {
			node = &NodeCapacity{NodeID: nodeID, HostProfile: hostProfile, PoolID: poolID}
			nodeIndex[nodeID] = node
		}
		node.WorkerCount++
		node.ParallelSlots += parallelSlots
		node.ActiveParallelSlots += activeSlots

		host := hostIndex[hostProfile]
		if host == nil {
			host = &hostProfileAccumulator{
				pools: map[string]struct{}{},
				nodes: map[string]struct{}{},
				HostProfileCapacity: HostProfileCapacity{
					HostProfile: hostProfile,
				},
			}
			hostIndex[hostProfile] = host
		}
		host.WorkerCount++
		host.ParallelSlots += parallelSlots
		host.ActiveParallelSlots += activeSlots
		host.pools[poolID] = struct{}{}
		host.nodes[nodeID] = struct{}{}

		pool := poolIndex[poolID]
		if pool == nil {
			pool = &poolAccumulator{
				hostProfiles: map[string]struct{}{},
				nodes:        map[string]struct{}{},
				PoolCapacity: PoolCapacity{
					PoolID: poolID,
				},
			}
			poolIndex[poolID] = pool
		}
		pool.WorkerCount++
		pool.ParallelSlots += parallelSlots
		pool.ActiveParallelSlots += activeSlots
		pool.hostProfiles[hostProfile] = struct{}{}
		pool.nodes[nodeID] = struct{}{}
	}

	baseline.AvailableParallelSlots = baseline.TotalParallelSlots - baseline.ActiveParallelSlots

	for _, node := range nodeIndex {
		node.AvailableParallelSlots = node.ParallelSlots - node.ActiveParallelSlots
		if node.ParallelSlots > 0 {
			node.CapacityUtilizationPercent = float64(node.ActiveParallelSlots) / float64(node.ParallelSlots) * 100
		}
		baseline.Nodes = append(baseline.Nodes, *node)
	}
	sort.SliceStable(baseline.Nodes, func(i, j int) bool { return baseline.Nodes[i].NodeID < baseline.Nodes[j].NodeID })

	for _, host := range hostIndex {
		host.NodeCount = len(host.nodes)
		host.PoolCount = len(host.pools)
		host.AvailableParallelSlots = host.ParallelSlots - host.ActiveParallelSlots
		if host.ParallelSlots > 0 {
			host.CapacityUtilizationPercent = float64(host.ActiveParallelSlots) / float64(host.ParallelSlots) * 100
		}
		baseline.HostProfiles = append(baseline.HostProfiles, host.HostProfileCapacity)
	}
	sort.SliceStable(baseline.HostProfiles, func(i, j int) bool {
		return baseline.HostProfiles[i].HostProfile < baseline.HostProfiles[j].HostProfile
	})

	for _, pool := range poolIndex {
		pool.HostProfileCount = len(pool.hostProfiles)
		pool.NodeCount = len(pool.nodes)
		pool.AvailableParallelSlots = pool.ParallelSlots - pool.ActiveParallelSlots
		if pool.ParallelSlots > 0 {
			pool.CapacityUtilizationPercent = float64(pool.ActiveParallelSlots) / float64(pool.ParallelSlots) * 100
		}
		baseline.Pools = append(baseline.Pools, pool.PoolCapacity)
	}
	sort.SliceStable(baseline.Pools, func(i, j int) bool { return baseline.Pools[i].PoolID < baseline.Pools[j].PoolID })

	return baseline
}

func isActiveState(state string) bool {
	switch strings.TrimSpace(state) {
	case "leased", "running":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
