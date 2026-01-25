package historical

import (
	"fmt"
	"math"
	"time"
)

// TrendDirection represents the direction of a trend
type TrendDirection int

const (
	// TrendUnknown indicates the trend direction is unknown.
	TrendUnknown TrendDirection = iota
	// TrendIncreasing indicates the trend is increasing.
	TrendIncreasing
	// TrendDecreasing indicates the trend is decreasing.
	TrendDecreasing
	// TrendStable indicates the trend is stable.
	TrendStable
	// TrendVolatile indicates the trend is volatile.
	TrendVolatile
)

func (td TrendDirection) String() string {
	switch td {
	case TrendIncreasing:
		return "increasing"
	case TrendDecreasing:
		return "decreasing"
	case TrendStable:
		return "stable"
	case TrendVolatile:
		return "volatile"
	default:
		return "unknown"
	}
}

// TrendAnalysis contains trend analysis results
type TrendAnalysis struct {
	Direction     TrendDirection `json:"direction"`
	Slope         float64        `json:"slope"`
	Correlation   float64        `json:"correlation"`
	Volatility    float64        `json:"volatility"`
	StartValue    float64        `json:"start_value"`
	EndValue      float64        `json:"end_value"`
	PercentChange float64        `json:"percent_change"`
	DataPoints    int            `json:"data_points"`
	TimeSpan      time.Duration  `json:"time_span"`
	Confidence    float64        `json:"confidence"`
}

// AnomalyDetection contains anomaly detection results
type AnomalyDetection struct {
	Anomalies    []AnomalyPoint `json:"anomalies"`
	AnomalyCount int            `json:"anomaly_count"`
	AnomalyRate  float64        `json:"anomaly_rate"`
	Method       string         `json:"method"`
	Threshold    float64        `json:"threshold"`
	Sensitivity  float64        `json:"sensitivity"`
}

// AnomalyPoint represents a detected anomaly
type AnomalyPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Expected  float64   `json:"expected"`
	Deviation float64   `json:"deviation"`
	Severity  string    `json:"severity"`
}

// SeasonalAnalysis contains seasonal pattern analysis
type SeasonalAnalysis struct {
	HasSeasonality bool               `json:"has_seasonality"`
	Period         time.Duration      `json:"period"`
	Strength       float64            `json:"strength"`
	Patterns       map[string]float64 `json:"patterns"`
	PeakTimes      []time.Time        `json:"peak_times"`
	LowTimes       []time.Time        `json:"low_times"`
}

// Analyzer provides analysis capabilities for historical data
type Analyzer struct {
	collector *DataCollector
}

//nolint:revive // type alias for backward compatibility
type HistoricalAnalyzer = Analyzer

// NewHistoricalAnalyzer creates a new historical data analyzer
func NewHistoricalAnalyzer(collector *DataCollector) *Analyzer {
	return &Analyzer{
		collector: collector,
	}
}

// AnalyzeTrend performs trend analysis on historical data
func (ha *HistoricalAnalyzer) AnalyzeTrend(metricName string, duration time.Duration) (*TrendAnalysis, error) {
	end := time.Now()
	start := end.Add(-duration)

	series, err := ha.collector.GetHistoricalData(metricName, start, end)
	if err != nil {
		return nil, err
	}

	if len(series.DataPoints) < 2 {
		return nil, fmt.Errorf("insufficient data points for trend analysis")
	}

	// Convert data points to numeric values with timestamps
	var values []float64
	var timestamps []float64

	for _, dp := range series.DataPoints {
		if val, ok := convertToFloat64(dp.Value); ok {
			values = append(values, val)
			timestamps = append(timestamps, float64(dp.Timestamp.Unix()))
		}
	}

	if len(values) < 2 {
		return nil, fmt.Errorf("insufficient numeric data points for trend analysis")
	}

	// Calculate linear regression
	slope, correlation := linearRegression(timestamps, values)

	// Calculate volatility (standard deviation)
	volatility := standardDeviation(values)

	// Determine trend direction
	direction := determineTrendDirection(slope, correlation, volatility)

	// Calculate percent change
	percentChange := 0.0
	if values[0] != 0 {
		percentChange = ((values[len(values)-1] - values[0]) / values[0]) * 100
	}

	// Calculate confidence based on correlation and data points
	confidence := math.Abs(correlation) * (float64(len(values)) / (float64(len(values)) + 10))

	return &TrendAnalysis{
		Direction:     direction,
		Slope:         slope,
		Correlation:   correlation,
		Volatility:    volatility,
		StartValue:    values[0],
		EndValue:      values[len(values)-1],
		PercentChange: percentChange,
		DataPoints:    len(values),
		TimeSpan:      duration,
		Confidence:    confidence,
	}, nil
}

// DetectAnomalies detects anomalies in historical data using statistical methods
func (ha *HistoricalAnalyzer) DetectAnomalies(metricName string, duration time.Duration, sensitivity float64) (*AnomalyDetection, error) {
	if sensitivity <= 0 {
		sensitivity = 2.0 // Default: 2 standard deviations
	}

	end := time.Now()
	start := end.Add(-duration)

	series, err := ha.collector.GetHistoricalData(metricName, start, end)
	if err != nil {
		return nil, err
	}

	if len(series.DataPoints) < 10 {
		return nil, fmt.Errorf("insufficient data points for anomaly detection")
	}

	// Extract numeric values
	var values []float64
	var dataPoints []DataPoint

	for _, dp := range series.DataPoints {
		if val, ok := convertToFloat64(dp.Value); ok {
			values = append(values, val)
			dataPoints = append(dataPoints, dp)
		}
	}

	if len(values) < 10 {
		return nil, fmt.Errorf("insufficient numeric data points for anomaly detection")
	}

	// Calculate statistical measures
	mean := average(values)
	stdDev := standardDeviation(values)
	threshold := stdDev * sensitivity

	// Detect anomalies using z-score method
	var anomalies []AnomalyPoint

	for i, val := range values {
		deviation := math.Abs(val - mean)
		zScore := deviation / stdDev

		if zScore > sensitivity {
			severity := "low"
			if zScore > sensitivity*1.5 {
				severity = "medium"
			}
			if zScore > sensitivity*2 {
				severity = "high"
			}

			anomaly := AnomalyPoint{
				Timestamp: dataPoints[i].Timestamp,
				Value:     val,
				Expected:  mean,
				Deviation: deviation,
				Severity:  severity,
			}

			anomalies = append(anomalies, anomaly)
		}
	}

	anomalyRate := float64(len(anomalies)) / float64(len(values))

	return &AnomalyDetection{
		Anomalies:    anomalies,
		AnomalyCount: len(anomalies),
		AnomalyRate:  anomalyRate,
		Method:       "z-score",
		Threshold:    threshold,
		Sensitivity:  sensitivity,
	}, nil
}

// AnalyzeSeasonality detects seasonal patterns in historical data
func (ha *HistoricalAnalyzer) AnalyzeSeasonality(metricName string, duration time.Duration) (*SeasonalAnalysis, error) {
	end := time.Now()
	start := end.Add(-duration)

	series, err := ha.collector.GetHistoricalData(metricName, start, end)
	if err != nil {
		return nil, err
	}

	if len(series.DataPoints) < 50 {
		return nil, fmt.Errorf("insufficient data points for seasonality analysis")
	}

	// Group data by hour of day
	hourlyData := make(map[int][]float64)
	var allValues []float64

	for _, dp := range series.DataPoints {
		if val, ok := convertToFloat64(dp.Value); ok {
			hour := dp.Timestamp.Hour()
			hourlyData[hour] = append(hourlyData[hour], val)
			allValues = append(allValues, val)
		}
	}

	if len(allValues) < 50 {
		return nil, fmt.Errorf("insufficient numeric data for seasonality analysis")
	}

	// Calculate hourly averages
	hourlyAverages := make(map[string]float64)
	overallMean := average(allValues)

	var maxAvg, minAvg float64
	var peakHours, lowHours []int
	first := true

	for hour := 0; hour < 24; hour++ {
		if values, exists := hourlyData[hour]; exists && len(values) > 0 {
			avg := average(values)
			hourlyAverages[fmt.Sprintf("hour_%02d", hour)] = avg

			if first {
				maxAvg = avg
				minAvg = avg
				first = false
			}

			if avg > maxAvg {
				maxAvg = avg
				peakHours = []int{hour}
			} else if avg == maxAvg {
				peakHours = append(peakHours, hour)
			}

			if avg < minAvg {
				minAvg = avg
				lowHours = []int{hour}
			} else if avg == minAvg {
				lowHours = append(lowHours, hour)
			}
		}
	}

	// Calculate seasonality strength
	seasonalVariance := (maxAvg - minAvg) / overallMean
	hasSeasonality := seasonalVariance > 0.1 // 10% variation threshold

	// Convert hours to timestamps (using today as base)
	today := time.Now().Truncate(24 * time.Hour)
	var peakTimes, lowTimes []time.Time

	for _, hour := range peakHours {
		peakTimes = append(peakTimes, today.Add(time.Duration(hour)*time.Hour))
	}

	for _, hour := range lowHours {
		lowTimes = append(lowTimes, today.Add(time.Duration(hour)*time.Hour))
	}

	return &SeasonalAnalysis{
		HasSeasonality: hasSeasonality,
		Period:         24 * time.Hour, // Daily period
		Strength:       seasonalVariance,
		Patterns:       hourlyAverages,
		PeakTimes:      peakTimes,
		LowTimes:       lowTimes,
	}, nil
}

// CompareMetrics compares multiple metrics over a time period
func (ha *HistoricalAnalyzer) CompareMetrics(metricNames []string, duration time.Duration) (map[string]interface{}, error) {
	end := time.Now()
	start := end.Add(-duration)

	comparison := make(map[string]interface{})
	var allStats []map[string]interface{}

	for _, metricName := range metricNames {
		series, err := ha.collector.GetHistoricalData(metricName, start, end)
		if err != nil {
			comparison[metricName] = map[string]interface{}{
				"error": err.Error(),
			}
			continue
		}

		// Calculate basic statistics
		var values []float64
		for _, dp := range series.DataPoints {
			if val, ok := convertToFloat64(dp.Value); ok {
				values = append(values, val)
			}
		}

		if len(values) == 0 {
			comparison[metricName] = map[string]interface{}{
				"error": "no numeric data points",
			}
			continue
		}

		stats := map[string]interface{}{
			"metric":   metricName,
			"count":    len(values),
			"mean":     average(values),
			"min":      minimum(values),
			"max":      maximum(values),
			"std_dev":  standardDeviation(values),
			"variance": variance(values),
		}

		comparison[metricName] = stats
		allStats = append(allStats, stats)
	}

	// Add correlation analysis if we have multiple valid metrics
	if len(allStats) >= 2 {
		// Simple correlation between first two metrics with valid data
		correlations := make(map[string]float64)

		for i := 0; i < len(allStats); i++ {
			for j := i + 1; j < len(allStats); j++ {
				metric1 := allStats[i]["metric"].(string)
				metric2 := allStats[j]["metric"].(string)

				// Get data for correlation calculation
				series1, _ := ha.collector.GetHistoricalData(metric1, start, end)
				series2, _ := ha.collector.GetHistoricalData(metric2, start, end)

				if series1 != nil && series2 != nil {
					corr := calculateCorrelation(series1.DataPoints, series2.DataPoints)
					correlations[fmt.Sprintf("%s_vs_%s", metric1, metric2)] = corr
				}
			}
		}

		comparison["correlations"] = correlations
	}

	return comparison, nil
}

// Helper functions for statistical calculations

func linearRegression(x, y []float64) (slope, correlation float64) {
	if len(x) != len(y) || len(x) < 2 {
		return 0, 0
	}

	n := float64(len(x))
	sumX := sum(x)
	sumY := sum(y)
	sumXY := 0.0
	sumX2 := 0.0
	sumY2 := 0.0

	for i := 0; i < len(x); i++ {
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	// Calculate slope
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, 0
	}
	slope = (n*sumXY - sumX*sumY) / denominator

	// Calculate correlation coefficient
	numerator := n*sumXY - sumX*sumY
	denominatorCorr := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))
	if denominatorCorr == 0 {
		correlation = 0
	} else {
		correlation = numerator / denominatorCorr
	}

	return slope, correlation
}

func determineTrendDirection(slope, correlation, volatility float64) TrendDirection {
	absCorr := math.Abs(correlation)

	if absCorr < 0.3 {
		return TrendUnknown
	}

	if volatility > 0.5 { // High volatility threshold
		return TrendVolatile
	}

	if absCorr > 0.7 { // Strong correlation
		if slope > 0 {
			return TrendIncreasing
		}
		return TrendDecreasing
	}

	return TrendStable
}

func calculateCorrelation(data1, data2 []DataPoint) float64 {
	// Align data points by timestamp
	var values1, values2 []float64

	// Create maps for fast lookup
	map1 := make(map[int64]float64)
	for _, dp := range data1 {
		if val, ok := convertToFloat64(dp.Value); ok {
			map1[dp.Timestamp.Unix()] = val
		}
	}

	// Find matching timestamps
	for _, dp := range data2 {
		if val, ok := convertToFloat64(dp.Value); ok {
			if val1, exists := map1[dp.Timestamp.Unix()]; exists {
				values1 = append(values1, val1)
				values2 = append(values2, val)
			}
		}
	}

	if len(values1) < 2 {
		return 0
	}

	// Calculate Pearson correlation
	_, correlation := linearRegression(values1, values2)
	return correlation
}

// Basic statistical functions
func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return sum(values) / float64(len(values))
}

func minimum(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func maximum(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func variance(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	mean := average(values)
	sumSquaredDiff := 0.0

	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}

	return sumSquaredDiff / float64(len(values)-1)
}

func standardDeviation(values []float64) float64 {
	return math.Sqrt(variance(values))
}
