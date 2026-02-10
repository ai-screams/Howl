package internal

import (
	"fmt"
	"testing"
)

// Helper: compare *int pointers
func intPtrEq(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// Helper: compare *float64 pointers
func floatPtrEq(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// Helper: create int pointer
func intPtr(v int) *int {
	return &v
}

// Helper: create float64 pointer
func floatPtr(v float64) *float64 {
	return &v
}

func TestCalcContextPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data *StdinData
		want int
	}{
		{
			name: "uses UsedPercentage when provided",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage: floatPtr(45.7),
				},
			},
			want: 45,
		},
		{
			name: "caps UsedPercentage at 100 when >100",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage: floatPtr(123.5),
				},
			},
			want: 100,
		},
		{
			name: "exactly 100",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage: floatPtr(100.0),
				},
			},
			want: 100,
		},
		{
			name: "fallback to CurrentUsage computation when UsedPercentage nil",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    nil,
					ContextWindowSize: 200000,
					CurrentUsage: &CurrentUsage{
						InputTokens:              10000,
						CacheCreationInputTokens: 5000,
						CacheReadInputTokens:     15000,
					},
				},
			},
			want: 15, // (10000+5000+15000)*100/200000 = 15
		},
		{
			name: "fallback returns 0 when CurrentUsage nil",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    nil,
					ContextWindowSize: 200000,
					CurrentUsage:      nil,
				},
			},
			want: 0,
		},
		{
			name: "fallback returns 0 when ContextWindowSize is 0",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    nil,
					ContextWindowSize: 0,
					CurrentUsage: &CurrentUsage{
						InputTokens: 10000,
					},
				},
			},
			want: 0,
		},
		{
			name: "fallback caps at 100 when computed >100",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    nil,
					ContextWindowSize: 100000,
					CurrentUsage: &CurrentUsage{
						InputTokens:              80000,
						CacheCreationInputTokens: 30000,
						CacheReadInputTokens:     5000,
					},
				},
			},
			want: 100, // (80000+30000+5000)*100/100000 = 115 → 100
		},
		{
			name: "fallback exact 100%",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    nil,
					ContextWindowSize: 100000,
					CurrentUsage: &CurrentUsage{
						InputTokens:              50000,
						CacheCreationInputTokens: 30000,
						CacheReadInputTokens:     20000,
					},
				},
			},
			want: 100,
		},
		{
			name: "zero tokens",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    nil,
					ContextWindowSize: 200000,
					CurrentUsage: &CurrentUsage{
						InputTokens:              0,
						CacheCreationInputTokens: 0,
						CacheReadInputTokens:     0,
					},
				},
			},
			want: 0,
		},
		{
			name: "1M context window fallback",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    nil,
					ContextWindowSize: 1000000,
					CurrentUsage: &CurrentUsage{
						InputTokens:              100000,
						CacheCreationInputTokens: 50000,
						CacheReadInputTokens:     350000,
					},
				},
			},
			want: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := calcContextPercent(tt.data)
			if got != tt.want {
				t.Errorf("calcContextPercent() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCalcCacheEfficiency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cu   *CurrentUsage
		want *int
	}{
		{
			name: "normal calculation",
			cu: &CurrentUsage{
				InputTokens:              10000,
				CacheCreationInputTokens: 5000,
				CacheReadInputTokens:     15000,
			},
			want: intPtr(50), // 15000*100/(10000+5000+15000) = 50
		},
		{
			name: "all zeros returns nil",
			cu: &CurrentUsage{
				InputTokens:              0,
				CacheCreationInputTokens: 0,
				CacheReadInputTokens:     0,
			},
			want: nil,
		},
		{
			name: "only cache_read returns 100%",
			cu: &CurrentUsage{
				InputTokens:              0,
				CacheCreationInputTokens: 0,
				CacheReadInputTokens:     10000,
			},
			want: intPtr(100),
		},
		{
			name: "only input tokens returns 0%",
			cu: &CurrentUsage{
				InputTokens:              10000,
				CacheCreationInputTokens: 0,
				CacheReadInputTokens:     0,
			},
			want: intPtr(0),
		},
		{
			name: "only cache_create returns 0%",
			cu: &CurrentUsage{
				InputTokens:              0,
				CacheCreationInputTokens: 10000,
				CacheReadInputTokens:     0,
			},
			want: intPtr(0),
		},
		{
			name: "large values no overflow",
			cu: &CurrentUsage{
				InputTokens:              50000,
				CacheCreationInputTokens: 80000,
				CacheReadInputTokens:     120000,
			},
			want: intPtr(48), // 120000*100/250000 = 48
		},
		{
			name: "exact 50%",
			cu: &CurrentUsage{
				InputTokens:              0,
				CacheCreationInputTokens: 10000,
				CacheReadInputTokens:     10000,
			},
			want: intPtr(50),
		},
		{
			name: "truncation test (66.66%)",
			cu: &CurrentUsage{
				InputTokens:              10000,
				CacheCreationInputTokens: 0,
				CacheReadInputTokens:     20000,
			},
			want: intPtr(66), // 20000*100/30000 = 66.666... → 66
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := calcCacheEfficiency(tt.cu)
			if !intPtrEq(got, tt.want) {
				t.Errorf("calcCacheEfficiency() = %s, want %s", ptrIntToString(got), ptrIntToString(tt.want))
			}
		})
	}
}

func TestCalcAPIWaitRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cost *Cost
		want *int
	}{
		{
			name: "normal ratio",
			cost: &Cost{
				TotalDurationMS:    10000,
				TotalAPIDurationMS: 2500,
			},
			want: intPtr(25), // 2500*100/10000 = 25
		},
		{
			name: "TotalDurationMS zero returns nil",
			cost: &Cost{
				TotalDurationMS:    0,
				TotalAPIDurationMS: 1000,
			},
			want: nil,
		},
		{
			name: "API duration greater than total (edge case)",
			cost: &Cost{
				TotalDurationMS:    5000,
				TotalAPIDurationMS: 8000,
			},
			want: intPtr(160), // 8000*100/5000 = 160 (can happen with measurement issues)
		},
		{
			name: "zero API duration returns 0%",
			cost: &Cost{
				TotalDurationMS:    10000,
				TotalAPIDurationMS: 0,
			},
			want: intPtr(0),
		},
		{
			name: "both zero returns nil",
			cost: &Cost{
				TotalDurationMS:    0,
				TotalAPIDurationMS: 0,
			},
			want: nil,
		},
		{
			name: "exact 100%",
			cost: &Cost{
				TotalDurationMS:    5000,
				TotalAPIDurationMS: 5000,
			},
			want: intPtr(100),
		},
		{
			name: "large values",
			cost: &Cost{
				TotalDurationMS:    3600000, // 1 hour
				TotalAPIDurationMS: 900000,  // 15 minutes
			},
			want: intPtr(25),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := calcAPIWaitRatio(tt.cost)
			if !intPtrEq(got, tt.want) {
				t.Errorf("calcAPIWaitRatio() = %s, want %s", ptrIntToString(got), ptrIntToString(tt.want))
			}
		})
	}
}

func TestCalcCostPerMinute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cost *Cost
		want *float64
	}{
		{
			name: "duration < 60000ms returns nil",
			cost: &Cost{
				TotalDurationMS: 59999,
				TotalCostUSD:    1.5,
			},
			want: nil,
		},
		{
			name: "exactly 60000ms (1 minute)",
			cost: &Cost{
				TotalDurationMS: 60000,
				TotalCostUSD:    1.2,
			},
			want: floatPtr(1.2),
		},
		{
			name: "normal calculation (2 minutes)",
			cost: &Cost{
				TotalDurationMS: 120000,
				TotalCostUSD:    2.4,
			},
			want: floatPtr(1.2),
		},
		{
			name: "zero cost returns 0.0/min",
			cost: &Cost{
				TotalDurationMS: 60000,
				TotalCostUSD:    0.0,
			},
			want: floatPtr(0.0),
		},
		{
			name: "fractional minutes",
			cost: &Cost{
				TotalDurationMS: 90000, // 1.5 minutes
				TotalCostUSD:    3.0,
			},
			want: floatPtr(2.0), // 3.0 / 1.5 = 2.0
		},
		{
			name: "large duration (1 hour)",
			cost: &Cost{
				TotalDurationMS: 3600000,
				TotalCostUSD:    12.0,
			},
			want: floatPtr(0.2), // 12.0 / 60 = 0.2
		},
		{
			name: "high cost velocity",
			cost: &Cost{
				TotalDurationMS: 60000,
				TotalCostUSD:    50.0,
			},
			want: floatPtr(50.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := calcCostPerMinute(tt.cost)
			if !floatPtrEq(got, tt.want) {
				t.Errorf("calcCostPerMinute() = %s, want %s", ptrFloatToString(got), ptrFloatToString(tt.want))
			}
		})
	}
}

func TestCalcResponseSpeed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cost *Cost
		cw   *ContextWindow
		want *int
	}{
		{
			name: "normal calculation",
			cost: &Cost{
				TotalAPIDurationMS: 5000, // 5 seconds
			},
			cw: &ContextWindow{
				TotalOutputTokens: 100,
			},
			want: intPtr(20), // 100 / 5.0 = 20 tok/s
		},
		{
			name: "TotalAPIDurationMS zero returns nil",
			cost: &Cost{
				TotalAPIDurationMS: 0,
			},
			cw: &ContextWindow{
				TotalOutputTokens: 100,
			},
			want: nil,
		},
		{
			name: "TotalOutputTokens zero returns nil",
			cost: &Cost{
				TotalAPIDurationMS: 5000,
			},
			cw: &ContextWindow{
				TotalOutputTokens: 0,
			},
			want: nil,
		},
		{
			name: "both zero returns nil",
			cost: &Cost{
				TotalAPIDurationMS: 0,
			},
			cw: &ContextWindow{
				TotalOutputTokens: 0,
			},
			want: nil,
		},
		{
			name: "large values",
			cost: &Cost{
				TotalAPIDurationMS: 10000, // 10 seconds
			},
			cw: &ContextWindow{
				TotalOutputTokens: 500,
			},
			want: intPtr(50), // 500 / 10.0 = 50 tok/s
		},
		{
			name: "fast response (100 tok/s)",
			cost: &Cost{
				TotalAPIDurationMS: 1000, // 1 second
			},
			cw: &ContextWindow{
				TotalOutputTokens: 100,
			},
			want: intPtr(100),
		},
		{
			name: "slow response (1 tok/s)",
			cost: &Cost{
				TotalAPIDurationMS: 10000, // 10 seconds
			},
			cw: &ContextWindow{
				TotalOutputTokens: 10,
			},
			want: intPtr(1),
		},
		{
			name: "truncation test (fractional tok/s)",
			cost: &Cost{
				TotalAPIDurationMS: 3000, // 3 seconds
			},
			cw: &ContextWindow{
				TotalOutputTokens: 100,
			},
			want: intPtr(33), // 100 / 3.0 = 33.333... → 33
		},
		{
			name: "very fast (sub-second)",
			cost: &Cost{
				TotalAPIDurationMS: 500, // 0.5 seconds
			},
			cw: &ContextWindow{
				TotalOutputTokens: 100,
			},
			want: intPtr(200), // 100 / 0.5 = 200 tok/s
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := calcResponseSpeed(tt.cost, tt.cw)
			if !intPtrEq(got, tt.want) {
				t.Errorf("calcResponseSpeed() = %s, want %s", ptrIntToString(got), ptrIntToString(tt.want))
			}
		})
	}
}

func TestComputeMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data *StdinData
		want Metrics
	}{
		{
			name: "all fields populated",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    floatPtr(45.7),
					ContextWindowSize: 200000,
					TotalOutputTokens: 100,
					CurrentUsage: &CurrentUsage{
						InputTokens:              10000,
						CacheCreationInputTokens: 5000,
						CacheReadInputTokens:     15000,
					},
				},
				Cost: Cost{
					TotalDurationMS:    120000, // 2 minutes
					TotalAPIDurationMS: 5000,
					TotalCostUSD:       2.4,
				},
			},
			want: Metrics{
				ContextPercent:  45,
				CacheEfficiency: intPtr(50),    // 15000*100/(10000+5000+15000) = 50
				APIWaitRatio:    intPtr(4),     // 5000*100/120000 = 4
				CostPerMinute:   floatPtr(1.2), // 2.4 / 2 = 1.2
				ResponseSpeed:   intPtr(20),    // 100 / 5.0 = 20
			},
		},
		{
			name: "minimal fields (nil CurrentUsage)",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    floatPtr(10.0),
					ContextWindowSize: 200000,
					TotalOutputTokens: 50,
					CurrentUsage:      nil,
				},
				Cost: Cost{
					TotalDurationMS:    30000,
					TotalAPIDurationMS: 0,
					TotalCostUSD:       0.5,
				},
			},
			want: Metrics{
				ContextPercent:  10,
				CacheEfficiency: nil,       // CurrentUsage is nil
				APIWaitRatio:    intPtr(0), // 0 API duration
				CostPerMinute:   nil,       // < 60000ms
				ResponseSpeed:   nil,       // API duration is 0
			},
		},
		{
			name: "zero struct",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    nil,
					ContextWindowSize: 0,
					TotalOutputTokens: 0,
					CurrentUsage:      nil,
				},
				Cost: Cost{
					TotalDurationMS:    0,
					TotalAPIDurationMS: 0,
					TotalCostUSD:       0.0,
				},
			},
			want: Metrics{
				ContextPercent:  0,
				CacheEfficiency: nil,
				APIWaitRatio:    nil,
				CostPerMinute:   nil,
				ResponseSpeed:   nil,
			},
		},
		{
			name: "high context usage with cache",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    floatPtr(87.3),
					ContextWindowSize: 200000,
					TotalOutputTokens: 500,
					CurrentUsage: &CurrentUsage{
						InputTokens:              30000,
						CacheCreationInputTokens: 10000,
						CacheReadInputTokens:     135000,
					},
				},
				Cost: Cost{
					TotalDurationMS:    300000, // 5 minutes
					TotalAPIDurationMS: 45000,
					TotalCostUSD:       15.0,
				},
			},
			want: Metrics{
				ContextPercent:  87,
				CacheEfficiency: intPtr(77),    // 135000*100/(30000+10000+135000) ≈ 77
				APIWaitRatio:    intPtr(15),    // 45000*100/300000 = 15
				CostPerMinute:   floatPtr(3.0), // 15.0 / 5 = 3.0
				ResponseSpeed:   intPtr(11),    // 500 / 45.0 ≈ 11
			},
		},
		{
			name: "exactly at thresholds",
			data: &StdinData{
				ContextWindow: ContextWindow{
					UsedPercentage:    floatPtr(100.0),
					ContextWindowSize: 200000,
					TotalOutputTokens: 120,
					CurrentUsage: &CurrentUsage{
						InputTokens:              0,
						CacheCreationInputTokens: 0,
						CacheReadInputTokens:     10000,
					},
				},
				Cost: Cost{
					TotalDurationMS:    60000, // exactly 1 minute
					TotalAPIDurationMS: 60000, // API = total
					TotalCostUSD:       5.0,
				},
			},
			want: Metrics{
				ContextPercent:  100,
				CacheEfficiency: intPtr(100),   // only cache_read
				APIWaitRatio:    intPtr(100),   // 100% API time
				CostPerMinute:   floatPtr(5.0), // 5.0 / 1 = 5.0
				ResponseSpeed:   intPtr(2),     // 120 / 60.0 = 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ComputeMetrics(tt.data)

			// Compare each field
			if got.ContextPercent != tt.want.ContextPercent {
				t.Errorf("ContextPercent = %d, want %d", got.ContextPercent, tt.want.ContextPercent)
			}
			if !intPtrEq(got.CacheEfficiency, tt.want.CacheEfficiency) {
				t.Errorf("CacheEfficiency = %v, want %v", ptrIntToString(got.CacheEfficiency), ptrIntToString(tt.want.CacheEfficiency))
			}
			if !intPtrEq(got.APIWaitRatio, tt.want.APIWaitRatio) {
				t.Errorf("APIWaitRatio = %v, want %v", ptrIntToString(got.APIWaitRatio), ptrIntToString(tt.want.APIWaitRatio))
			}
			if !floatPtrEq(got.CostPerMinute, tt.want.CostPerMinute) {
				t.Errorf("CostPerMinute = %v, want %v", ptrFloatToString(got.CostPerMinute), ptrFloatToString(tt.want.CostPerMinute))
			}
			if !intPtrEq(got.ResponseSpeed, tt.want.ResponseSpeed) {
				t.Errorf("ResponseSpeed = %v, want %v", ptrIntToString(got.ResponseSpeed), ptrIntToString(tt.want.ResponseSpeed))
			}
		})
	}
}

// Helper for printing pointer values in errors
func ptrIntToString(p *int) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *p)
}

// Helper for printing pointer values in errors
func ptrFloatToString(p *float64) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%.2f", *p)
}
