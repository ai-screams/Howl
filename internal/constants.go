package internal

import "time"

// Context and layout thresholds
const (
	DangerThreshold   = 85 // Context percentage to trigger danger mode
	WarningThreshold  = 70 // Show warning indicator
	ModerateThreshold = 50 // Context % for yellow (moderate usage)
)

// Cache efficiency thresholds (percentage)
const (
	CacheExcellent = 80 // Green: excellent cache utilization
	CacheGood      = 50 // Yellow: moderate cache usage
	// Below 50% = Red: poor cache utilization
)

// API wait ratio thresholds (percentage)
const (
	WaitHigh   = 60 // Red: most time spent waiting
	WaitMedium = 35 // Yellow: balanced wait time
	// Below 35% = Green: minimal wait time
)

// Response speed thresholds (tokens/second)
const (
	SpeedFast     = 60 // Green: fast response
	SpeedModerate = 30 // Yellow: moderate speed
	// Below 30 = Orange: slow response
)

// Cost velocity thresholds ($/minute)
const (
	CostHigh   = 0.50 // Red: expensive session
	CostMedium = 0.10 // Yellow: moderate cost
	// Below 0.10 = Green: economical
)

// Session cost thresholds (USD)
const (
	SessionCostHigh   = 5.0 // Red: expensive session
	SessionCostMedium = 1.0 // Yellow: moderate session
	// Below 1.0 = White: normal cost
)

// Time conversion constants
const msPerMinute = 60000 // milliseconds in one minute

// Usage quota thresholds (percentage remaining)
const (
	QuotaCritical = 10 // Red bold: almost depleted
	QuotaLow      = 25 // Red: low quota
	QuotaMedium   = 50 // Orange: half used
	QuotaHigh     = 75 // Yellow: comfortable
	// Above 75% = Green: plenty remaining
)

// External operation timeouts and cache TTL
const (
	GitTimeout      = 1 * time.Second  // Git subprocess timeout
	UsageCacheTTL   = 60 * time.Second // OAuth quota cache duration
	UsageAPITimeout = 3 * time.Second  // OAuth API request timeout
)
