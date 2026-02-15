package internal

// Metrics holds all derived values computed from StdinData.
type Metrics struct {
	ContextPercent  int  // 0-100
	CacheEfficiency *int // nil if insufficient data
	APIWaitRatio    *int // nil if duration=0
	CostPerMinute   *float64
	ResponseSpeed   *int // tokens/sec output speed
}

// ComputeMetrics calculates derived KPIs from raw session data.
// It computes context percentage, cache efficiency, API wait ratio, cost velocity, and response speed.
func ComputeMetrics(d *StdinData) Metrics {
	m := Metrics{
		ContextPercent: calcContextPercent(d),
	}
	if d.ContextWindow.CurrentUsage != nil {
		m.CacheEfficiency = calcCacheEfficiency(d.ContextWindow.CurrentUsage)
	}
	m.APIWaitRatio = calcAPIWaitRatio(&d.Cost)
	m.CostPerMinute = calcCostPerMinute(&d.Cost)
	m.ResponseSpeed = calcResponseSpeed(&d.Cost, &d.ContextWindow)
	return m
}

func calcContextPercent(d *StdinData) int {
	if d.ContextWindow.UsedPercentage != nil {
		p := int(*d.ContextWindow.UsedPercentage)
		if p > 100 {
			return 100
		}
		return p
	}
	// Fallback: compute from current_usage tokens
	cu := d.ContextWindow.CurrentUsage
	if cu == nil || d.ContextWindow.ContextWindowSize == 0 {
		return 0
	}
	total := cu.InputTokens + cu.CacheCreationInputTokens + cu.CacheReadInputTokens
	p := total * 100 / d.ContextWindow.ContextWindowSize
	if p > 100 {
		return 100
	}
	return p
}

func calcCacheEfficiency(cu *CurrentUsage) *int {
	total := cu.CacheReadInputTokens + cu.CacheCreationInputTokens + cu.InputTokens
	if total == 0 {
		return nil
	}
	v := cu.CacheReadInputTokens * 100 / total
	return &v
}

func calcAPIWaitRatio(c *Cost) *int {
	if c.TotalDurationMS == 0 {
		return nil
	}
	v := int(c.TotalAPIDurationMS * 100 / c.TotalDurationMS)
	return &v
}

func calcCostPerMinute(c *Cost) *float64 {
	if c.TotalDurationMS < msPerMinute { // need at least 1 minute
		return nil
	}
	minutes := float64(c.TotalDurationMS) / msPerMinute
	v := c.TotalCostUSD / minutes
	return &v
}

func calcResponseSpeed(c *Cost, cw *ContextWindow) *int {
	if c.TotalAPIDurationMS == 0 || cw.TotalOutputTokens == 0 {
		return nil
	}
	seconds := float64(c.TotalAPIDurationMS) / 1000.0
	speed := float64(cw.TotalOutputTokens) / seconds
	v := int(speed)
	return &v
}
