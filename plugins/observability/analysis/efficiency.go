package analysis

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jontk/s9s/plugins/observability/historical"
	"github.com/jontk/s9s/plugins/observability/prometheus"
)

// ResourceType represents different types of cluster resources
type ResourceType string

const (
	ResourceCPU        ResourceType = "cpu"
	ResourceMemory     ResourceType = "memory"
	ResourceStorage    ResourceType = "storage"
	ResourceNetwork    ResourceType = "network"
	ResourceGPU        ResourceType = "gpu"
)

// EfficiencyLevel represents efficiency rating levels
type EfficiencyLevel string

const (
	EfficiencyExcellent EfficiencyLevel = "excellent"
	EfficiencyGood      EfficiencyLevel = "good"
	EfficiencyFair      EfficiencyLevel = "fair"
	EfficiencyPoor      EfficiencyLevel = "poor"
	EfficiencyCritical  EfficiencyLevel = "critical"
)

// ResourceEfficiency represents efficiency analysis for a resource type
type ResourceEfficiency struct {
	ResourceType     ResourceType      `json:"resource_type"`
	OverallScore     float64           `json:"overall_score"`
	EfficiencyLevel  EfficiencyLevel   `json:"efficiency_level"`
	Utilization      UtilizationStats  `json:"utilization"`
	Recommendations  []Recommendation  `json:"recommendations"`
	TrendAnalysis    *TrendSummary     `json:"trend_analysis,omitempty"`
	CostImpact       *CostAnalysis     `json:"cost_impact,omitempty"`
	LastAnalyzed     time.Time         `json:"last_analyzed"`
}

// UtilizationStats contains utilization statistics
type UtilizationStats struct {
	Average           float64   `json:"average"`
	Peak              float64   `json:"peak"`
	Low               float64   `json:"low"`
	StandardDeviation float64   `json:"standard_deviation"`
	Percentiles       Percentiles `json:"percentiles"`
	WastePercentage   float64   `json:"waste_percentage"`
	IdleTime          time.Duration `json:"idle_time"`
}

// Percentiles contains various percentile values
type Percentiles struct {
	P50 float64 `json:"p50"`
	P75 float64 `json:"p75"`
	P90 float64 `json:"p90"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// Recommendation represents an optimization recommendation
type Recommendation struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Impact         RecommendationImpact `json:"impact"`
	Confidence     float64         `json:"confidence"`
	EstimatedSaving float64        `json:"estimated_saving"`
	ImplementationComplexity string `json:"implementation_complexity"`
	Priority       int             `json:"priority"`
	Category       string          `json:"category"`
}

// RecommendationImpact describes the potential impact of a recommendation
type RecommendationImpact struct {
	ResourceSaving   float64 `json:"resource_saving"`
	CostSaving       float64 `json:"cost_saving"`
	PerformanceGain  float64 `json:"performance_gain"`
	EfficiencyGain   float64 `json:"efficiency_gain"`
}

// TrendSummary provides a summary of trend analysis
type TrendSummary struct {
	Direction    string    `json:"direction"`
	Slope        float64   `json:"slope"`
	Confidence   float64   `json:"confidence"`
	Prediction   float64   `json:"prediction"`
	PredictionDate time.Time `json:"prediction_date"`
}

// CostAnalysis provides cost-related analysis
type CostAnalysis struct {
	CurrentCost      float64 `json:"current_cost"`
	OptimizedCost    float64 `json:"optimized_cost"`
	PotentialSaving  float64 `json:"potential_saving"`
	ROI              float64 `json:"roi"`
	PaybackPeriod    time.Duration `json:"payback_period"`
}

// ClusterEfficiencyAnalysis contains overall cluster efficiency analysis
type ClusterEfficiencyAnalysis struct {
	OverallScore         float64                        `json:"overall_score"`
	EfficiencyLevel      EfficiencyLevel               `json:"efficiency_level"`
	ResourceAnalysis     map[ResourceType]*ResourceEfficiency `json:"resource_analysis"`
	TopRecommendations   []Recommendation              `json:"top_recommendations"`
	EfficiencyTrends     map[ResourceType]*TrendSummary `json:"efficiency_trends"`
	ClusterUtilization   ClusterUtilizationSummary     `json:"cluster_utilization"`
	CostOptimization     *CostOptimizationSummary      `json:"cost_optimization,omitempty"`
	LastAnalyzed         time.Time                     `json:"last_analyzed"`
	AnalysisPeriod       time.Duration                 `json:"analysis_period"`
}

// ClusterUtilizationSummary provides cluster-wide utilization summary
type ClusterUtilizationSummary struct {
	TotalNodes       int             `json:"total_nodes"`
	ActiveNodes      int             `json:"active_nodes"`
	IdleNodes        int             `json:"idle_nodes"`
	TotalJobs        int             `json:"total_jobs"`
	QueuedJobs       int             `json:"queued_jobs"`
	RunningJobs      int             `json:"running_jobs"`
	AverageWaitTime  time.Duration   `json:"average_wait_time"`
	ResourceWaste    map[ResourceType]float64 `json:"resource_waste"`
}

// CostOptimizationSummary provides cost optimization insights
type CostOptimizationSummary struct {
	CurrentMonthlyCost   float64 `json:"current_monthly_cost"`
	OptimizedMonthlyCost float64 `json:"optimized_monthly_cost"`
	MonthlySavings       float64 `json:"monthly_savings"`
	AnnualSavings        float64 `json:"annual_savings"`
	OptimizationROI      float64 `json:"optimization_roi"`
}

// ResourceEfficiencyAnalyzer analyzes resource efficiency
type ResourceEfficiencyAnalyzer struct {
	client            *prometheus.CachedClient
	historicalCollector *historical.HistoricalDataCollector
	historicalAnalyzer  *historical.HistoricalAnalyzer
}

// NewResourceEfficiencyAnalyzer creates a new resource efficiency analyzer
func NewResourceEfficiencyAnalyzer(client *prometheus.CachedClient, collector *historical.HistoricalDataCollector, analyzer *historical.HistoricalAnalyzer) *ResourceEfficiencyAnalyzer {
	return &ResourceEfficiencyAnalyzer{
		client:              client,
		historicalCollector: collector,
		historicalAnalyzer:  analyzer,
	}
}

// AnalyzeClusterEfficiency performs comprehensive cluster efficiency analysis
func (rea *ResourceEfficiencyAnalyzer) AnalyzeClusterEfficiency(ctx context.Context, analysisPeriod time.Duration) (*ClusterEfficiencyAnalysis, error) {
	analysis := &ClusterEfficiencyAnalysis{
		ResourceAnalysis:   make(map[ResourceType]*ResourceEfficiency),
		EfficiencyTrends:   make(map[ResourceType]*TrendSummary),
		LastAnalyzed:       time.Now(),
		AnalysisPeriod:     analysisPeriod,
	}

	// Analyze each resource type
	resourceTypes := []ResourceType{ResourceCPU, ResourceMemory, ResourceStorage, ResourceNetwork}
	
	var totalScore float64
	var allRecommendations []Recommendation

	for _, resourceType := range resourceTypes {
		resourceAnalysis, err := rea.AnalyzeResourceEfficiency(ctx, resourceType, analysisPeriod)
		if err != nil {
			// Log error but continue with other resources
			continue
		}

		analysis.ResourceAnalysis[resourceType] = resourceAnalysis
		totalScore += resourceAnalysis.OverallScore
		allRecommendations = append(allRecommendations, resourceAnalysis.Recommendations...)

		// Add trend analysis
		if resourceAnalysis.TrendAnalysis != nil {
			analysis.EfficiencyTrends[resourceType] = resourceAnalysis.TrendAnalysis
		}
	}

	// Calculate overall score
	if len(analysis.ResourceAnalysis) > 0 {
		analysis.OverallScore = totalScore / float64(len(analysis.ResourceAnalysis))
		analysis.EfficiencyLevel = rea.calculateEfficiencyLevel(analysis.OverallScore)
	}

	// Sort and select top recommendations
	sort.Slice(allRecommendations, func(i, j int) bool {
		return allRecommendations[i].Priority < allRecommendations[j].Priority
	})

	maxRecommendations := 10
	if len(allRecommendations) > maxRecommendations {
		analysis.TopRecommendations = allRecommendations[:maxRecommendations]
	} else {
		analysis.TopRecommendations = allRecommendations
	}

	// Analyze cluster utilization
	utilizationSummary, err := rea.analyzeClusterUtilization(ctx)
	if err == nil {
		analysis.ClusterUtilization = *utilizationSummary
	}

	// Calculate cost optimization if possible
	costOptimization, err := rea.calculateCostOptimization(analysis)
	if err == nil {
		analysis.CostOptimization = costOptimization
	}

	return analysis, nil
}

// AnalyzeResourceEfficiency analyzes efficiency for a specific resource type
func (rea *ResourceEfficiencyAnalyzer) AnalyzeResourceEfficiency(ctx context.Context, resourceType ResourceType, analysisPeriod time.Duration) (*ResourceEfficiency, error) {
	metricName := rea.getMetricName(resourceType)
	if metricName == "" {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	// Get historical data
	end := time.Now()
	start := end.Add(-analysisPeriod)

	series, err := rea.historicalCollector.GetHistoricalData(metricName, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical data: %w", err)
	}

	if len(series.DataPoints) < 10 {
		return nil, fmt.Errorf("insufficient data points for analysis")
	}

	// Calculate utilization statistics
	utilizationStats, err := rea.calculateUtilizationStats(series.DataPoints)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate utilization stats: %w", err)
	}

	// Calculate overall efficiency score
	overallScore := rea.calculateEfficiencyScore(utilizationStats, resourceType)
	
	// Generate recommendations
	recommendations := rea.generateRecommendations(resourceType, utilizationStats, overallScore)

	// Get trend analysis
	trendAnalysis, err := rea.historicalAnalyzer.AnalyzeTrend(metricName, analysisPeriod)
	var trendSummary *TrendSummary
	if err == nil {
		trendSummary = &TrendSummary{
			Direction:      trendAnalysis.Direction.String(),
			Slope:          trendAnalysis.Slope,
			Confidence:     trendAnalysis.Confidence,
			Prediction:     trendAnalysis.EndValue + (trendAnalysis.Slope * 86400), // 24h prediction
			PredictionDate: time.Now().Add(24 * time.Hour),
		}
	}

	// Calculate cost impact
	costImpact := rea.calculateCostImpact(resourceType, utilizationStats, recommendations)

	return &ResourceEfficiency{
		ResourceType:    resourceType,
		OverallScore:    overallScore,
		EfficiencyLevel: rea.calculateEfficiencyLevel(overallScore),
		Utilization:     *utilizationStats,
		Recommendations: recommendations,
		TrendAnalysis:   trendSummary,
		CostImpact:      costImpact,
		LastAnalyzed:    time.Now(),
	}, nil
}

// calculateUtilizationStats calculates utilization statistics from data points
func (rea *ResourceEfficiencyAnalyzer) calculateUtilizationStats(dataPoints []historical.DataPoint) (*UtilizationStats, error) {
	var values []float64
	var idleCount int

	for _, dp := range dataPoints {
		if val, ok := rea.convertToFloat64(dp.Value); ok {
			values = append(values, val)
			if val < 5.0 { // Consider < 5% as idle
				idleCount++
			}
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no valid numeric data points")
	}

	// Sort values for percentile calculations
	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	// Calculate basic statistics
	average := rea.average(values)
	peak := sortedValues[len(sortedValues)-1]
	low := sortedValues[0]
	stdDev := rea.standardDeviation(values)

	// Calculate percentiles
	percentiles := Percentiles{
		P50: rea.percentile(sortedValues, 0.50),
		P75: rea.percentile(sortedValues, 0.75),
		P90: rea.percentile(sortedValues, 0.90),
		P95: rea.percentile(sortedValues, 0.95),
		P99: rea.percentile(sortedValues, 0.99),
	}

	// Calculate waste percentage (resources allocated but not used effectively)
	wastePercentage := math.Max(0, 100-average)
	if average > 80 {
		wastePercentage = 0 // High utilization means low waste
	}

	// Calculate idle time
	totalDataPoints := len(dataPoints)
	idlePercentage := float64(idleCount) / float64(totalDataPoints)
	
	// Assuming data points are collected every 5 minutes
	totalDuration := time.Duration(totalDataPoints) * 5 * time.Minute
	idleTime := time.Duration(float64(totalDuration) * idlePercentage)

	return &UtilizationStats{
		Average:           average,
		Peak:              peak,
		Low:               low,
		StandardDeviation: stdDev,
		Percentiles:       percentiles,
		WastePercentage:   wastePercentage,
		IdleTime:          idleTime,
	}, nil
}

// calculateEfficiencyScore calculates an overall efficiency score (0-100)
func (rea *ResourceEfficiencyAnalyzer) calculateEfficiencyScore(stats *UtilizationStats, resourceType ResourceType) float64 {
	// Base score from utilization (optimal range: 70-85%)
	utilizationScore := 100.0
	if stats.Average < 20 {
		utilizationScore = stats.Average * 2 // Heavily penalize low utilization
	} else if stats.Average < 70 {
		utilizationScore = 40 + (stats.Average-20)*1.2 // Gradual penalty
	} else if stats.Average <= 85 {
		utilizationScore = 100 // Optimal range
	} else {
		utilizationScore = 100 - (stats.Average-85)*2 // Penalize over-utilization
	}

	// Stability score (lower standard deviation is better)
	stabilityScore := 100.0
	if stats.StandardDeviation > 30 {
		stabilityScore = 50 // High variability is bad
	} else if stats.StandardDeviation > 15 {
		stabilityScore = 70 + (30-stats.StandardDeviation)*2
	}

	// Waste penalty
	wasteScore := 100 - stats.WastePercentage

	// Resource-specific adjustments
	var resourceMultiplier float64 = 1.0
	switch resourceType {
	case ResourceCPU:
		// CPU efficiency is critical for performance
		resourceMultiplier = 1.1
	case ResourceMemory:
		// Memory efficiency affects stability
		resourceMultiplier = 1.05
	case ResourceStorage:
		// Storage efficiency affects cost
		resourceMultiplier = 1.0
	case ResourceNetwork:
		// Network efficiency affects throughput
		resourceMultiplier = 0.95
	}

	// Combine scores with weights
	overallScore := (utilizationScore*0.5 + stabilityScore*0.3 + wasteScore*0.2) * resourceMultiplier

	// Ensure score is between 0 and 100
	if overallScore > 100 {
		overallScore = 100
	} else if overallScore < 0 {
		overallScore = 0
	}

	return overallScore
}

// calculateEfficiencyLevel determines efficiency level from score
func (rea *ResourceEfficiencyAnalyzer) calculateEfficiencyLevel(score float64) EfficiencyLevel {
	switch {
	case score >= 90:
		return EfficiencyExcellent
	case score >= 75:
		return EfficiencyGood
	case score >= 60:
		return EfficiencyFair
	case score >= 40:
		return EfficiencyPoor
	default:
		return EfficiencyCritical
	}
}

// generateRecommendations generates optimization recommendations
func (rea *ResourceEfficiencyAnalyzer) generateRecommendations(resourceType ResourceType, stats *UtilizationStats, score float64) []Recommendation {
	var recommendations []Recommendation

	// Low utilization recommendations
	if stats.Average < 30 {
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("%s_low_utilization", resourceType),
			Title:       fmt.Sprintf("Low %s Utilization", resourceType),
			Description: fmt.Sprintf("Average %s utilization is %.1f%%, indicating over-provisioning", resourceType, stats.Average),
			Impact: RecommendationImpact{
				ResourceSaving: (50 - stats.Average) / 100,
				CostSaving:     (50 - stats.Average) * 2,
				EfficiencyGain: (50 - stats.Average) / 2,
			},
			Confidence:               85.0,
			EstimatedSaving:          (50 - stats.Average) * 2,
			ImplementationComplexity: "Medium",
			Priority:                 1,
			Category:                 "Right-sizing",
		})
	}

	// High variability recommendations
	if stats.StandardDeviation > 25 {
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("%s_high_variability", resourceType),
			Title:       fmt.Sprintf("High %s Variability", resourceType),
			Description: fmt.Sprintf("%s usage shows high variability (σ=%.1f), consider auto-scaling", resourceType, stats.StandardDeviation),
			Impact: RecommendationImpact{
				ResourceSaving:  0.1,
				PerformanceGain: 0.15,
				EfficiencyGain:  0.2,
			},
			Confidence:               75.0,
			EstimatedSaving:          10.0,
			ImplementationComplexity: "High",
			Priority:                 2,
			Category:                 "Auto-scaling",
		})
	}

	// Over-utilization recommendations
	if stats.Average > 90 {
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("%s_over_utilization", resourceType),
			Title:       fmt.Sprintf("High %s Utilization", resourceType),
			Description: fmt.Sprintf("Average %s utilization is %.1f%%, risking performance degradation", resourceType, stats.Average),
			Impact: RecommendationImpact{
				PerformanceGain: 0.25,
				EfficiencyGain:  0.15,
			},
			Confidence:               90.0,
			EstimatedSaving:          0,
			ImplementationComplexity: "Low",
			Priority:                 1,
			Category:                 "Capacity Planning",
		})
	}

	// Idle time recommendations
	if stats.IdleTime > time.Duration(float64(24*time.Hour)*0.3) { // More than 30% idle
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("%s_excessive_idle", resourceType),
			Title:       fmt.Sprintf("Excessive %s Idle Time", resourceType),
			Description: fmt.Sprintf("%s is idle for %v, consider consolidation or shutdown policies", resourceType, stats.IdleTime),
			Impact: RecommendationImpact{
				ResourceSaving: 0.3,
				CostSaving:     30.0,
				EfficiencyGain: 0.4,
			},
			Confidence:               80.0,
			EstimatedSaving:          25.0,
			ImplementationComplexity: "Medium",
			Priority:                 2,
			Category:                 "Power Management",
		})
	}

	return recommendations
}

// calculateCostImpact calculates cost impact analysis
func (rea *ResourceEfficiencyAnalyzer) calculateCostImpact(resourceType ResourceType, stats *UtilizationStats, recommendations []Recommendation) *CostAnalysis {
	// Simplified cost model - in production, this would use actual pricing data
	baseCostPerUnit := rea.getBaseCostPerUnit(resourceType)
	
	// Current cost based on average utilization
	currentCost := baseCostPerUnit * (stats.Average / 100)
	
	// Calculate potential optimized cost
	totalSaving := 0.0
	for _, rec := range recommendations {
		totalSaving += rec.EstimatedSaving
	}
	
	optimizedCost := currentCost * (1 - totalSaving/100)
	potentialSaving := currentCost - optimizedCost
	
	// Calculate ROI (assuming implementation cost is 10% of savings)
	implementationCost := potentialSaving * 0.1
	roi := 0.0
	if implementationCost > 0 {
		roi = (potentialSaving - implementationCost) / implementationCost * 100
	}
	
	// Payback period (simplified to 3-6 months based on savings)
	paybackPeriod := 6 * time.Hour * 24 * 30 // 6 months default
	if totalSaving > 20 {
		paybackPeriod = 3 * time.Hour * 24 * 30 // 3 months for high savings
	}

	return &CostAnalysis{
		CurrentCost:     currentCost,
		OptimizedCost:   optimizedCost,
		PotentialSaving: potentialSaving,
		ROI:             roi,
		PaybackPeriod:   paybackPeriod,
	}
}

// analyzeClusterUtilization analyzes overall cluster utilization
func (rea *ResourceEfficiencyAnalyzer) analyzeClusterUtilization(ctx context.Context) (*ClusterUtilizationSummary, error) {
	// This would typically query SLURM metrics for cluster-wide statistics
	// For now, we'll return a simplified analysis
	
	summary := &ClusterUtilizationSummary{
		TotalNodes:      10,  // This would be queried from SLURM
		ActiveNodes:     8,
		IdleNodes:       2,
		TotalJobs:       50,
		QueuedJobs:      5,
		RunningJobs:     45,
		AverageWaitTime: 10 * time.Minute,
		ResourceWaste:   make(map[ResourceType]float64),
	}

	// Calculate resource waste for each resource type
	resourceTypes := []ResourceType{ResourceCPU, ResourceMemory, ResourceStorage, ResourceNetwork}
	for _, resourceType := range resourceTypes {
		// This is a simplified calculation - would use actual metrics in production
		summary.ResourceWaste[resourceType] = 15.0 // 15% average waste
	}

	return summary, nil
}

// calculateCostOptimization calculates cost optimization summary
func (rea *ResourceEfficiencyAnalyzer) calculateCostOptimization(analysis *ClusterEfficiencyAnalysis) (*CostOptimizationSummary, error) {
	currentMonthlyCost := 10000.0 // This would be calculated from actual usage
	totalSavingsPercentage := 0.0
	
	// Sum up savings from all recommendations
	for _, rec := range analysis.TopRecommendations {
		totalSavingsPercentage += rec.EstimatedSaving
	}
	
	// Cap savings at reasonable maximum
	if totalSavingsPercentage > 40 {
		totalSavingsPercentage = 40
	}
	
	monthlySavings := currentMonthlyCost * (totalSavingsPercentage / 100)
	optimizedMonthlyCost := currentMonthlyCost - monthlySavings
	annualSavings := monthlySavings * 12
	
	// Calculate ROI based on implementation effort
	implementationCost := monthlySavings * 2 // Assume 2 months implementation cost
	roi := 0.0
	if implementationCost > 0 {
		roi = (annualSavings - implementationCost) / implementationCost * 100
	}

	return &CostOptimizationSummary{
		CurrentMonthlyCost:   currentMonthlyCost,
		OptimizedMonthlyCost: optimizedMonthlyCost,
		MonthlySavings:       monthlySavings,
		AnnualSavings:        annualSavings,
		OptimizationROI:      roi,
	}, nil
}

// Helper methods

func (rea *ResourceEfficiencyAnalyzer) getMetricName(resourceType ResourceType) string {
	switch resourceType {
	case ResourceCPU:
		return "node_cpu"
	case ResourceMemory:
		return "node_memory"
	case ResourceStorage:
		return "node_disk"
	case ResourceNetwork:
		return "node_network"
	default:
		return ""
	}
}

func (rea *ResourceEfficiencyAnalyzer) getBaseCostPerUnit(resourceType ResourceType) float64 {
	switch resourceType {
	case ResourceCPU:
		return 50.0 // $50 per CPU core per month
	case ResourceMemory:
		return 5.0  // $5 per GB per month
	case ResourceStorage:
		return 0.1  // $0.1 per GB per month
	case ResourceNetwork:
		return 10.0 // $10 per Gbps per month
	default:
		return 1.0
	}
}

func (rea *ResourceEfficiencyAnalyzer) convertToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}

func (rea *ResourceEfficiencyAnalyzer) average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (rea *ResourceEfficiencyAnalyzer) standardDeviation(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	mean := rea.average(values)
	sumSquaredDiff := 0.0
	
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	
	variance := sumSquaredDiff / float64(len(values)-1)
	return math.Sqrt(variance)
}

func (rea *ResourceEfficiencyAnalyzer) percentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	
	index := percentile * float64(len(sortedValues)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	
	if lower == upper {
		return sortedValues[lower]
	}
	
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}