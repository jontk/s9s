package dao

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeTolerance = 5 * time.Second

func TestParseBeginTime_NamedTimes(t *testing.T) {
	t.Run("now", func(t *testing.T) {
		before := time.Now()
		result, err := parseBeginTime("now")
		after := time.Now()
		require.NoError(t, err)
		ts := time.Unix(int64(result), 0)
		assert.True(t, !ts.Before(before.Add(-timeTolerance)), "timestamp should not be before now-tolerance")
		assert.True(t, !ts.After(after.Add(timeTolerance)), "timestamp should not be after now+tolerance")
	})

	t.Run("today", func(t *testing.T) {
		result, err := parseBeginTime("today")
		require.NoError(t, err)
		now := time.Now()
		expected := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		assert.Equal(t, uint64(expected.Unix()), result)
	})

	t.Run("tomorrow", func(t *testing.T) {
		result, err := parseBeginTime("tomorrow")
		require.NoError(t, err)
		now := time.Now()
		expected := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(24 * time.Hour)
		assert.Equal(t, uint64(expected.Unix()), result)
	})
}

func TestParseBeginTime_NamedDayTimes(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		name  string
		input string
		hour  int
	}{
		{"noon", "noon", 12},
		{"elevenses", "elevenses", 11},
		{"fika", "fika", 15},
		{"teatime", "teatime", 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseBeginTime(tt.input)
			require.NoError(t, err)

			expected := today.Add(time.Duration(tt.hour) * time.Hour)
			if now.After(expected) {
				expected = expected.Add(24 * time.Hour)
			}
			assert.Equal(t, uint64(expected.Unix()), result)
		})
	}

	t.Run("midnight", func(t *testing.T) {
		result, err := parseBeginTime("midnight")
		require.NoError(t, err)
		// midnight is next day's midnight (unless it's exactly midnight now)
		expected := today.Add(24 * time.Hour)
		if now.Before(today.Add(1 * time.Second)) {
			expected = today
		}
		assert.Equal(t, uint64(expected.Unix()), result)
	})
}

func TestParseBeginTime_Relative(t *testing.T) {
	t.Run("now+60 (seconds)", func(t *testing.T) {
		before := time.Now()
		result, err := parseBeginTime("now+60")
		require.NoError(t, err)
		expected := before.Add(60 * time.Second)
		ts := time.Unix(int64(result), 0)
		assert.WithinDuration(t, expected, ts, timeTolerance)
	})

	t.Run("now+1hour", func(t *testing.T) {
		before := time.Now()
		result, err := parseBeginTime("now+1hour")
		require.NoError(t, err)
		expected := before.Add(1 * time.Hour)
		ts := time.Unix(int64(result), 0)
		assert.WithinDuration(t, expected, ts, timeTolerance)
	})

	t.Run("now+30minutes", func(t *testing.T) {
		before := time.Now()
		result, err := parseBeginTime("now+30minutes")
		require.NoError(t, err)
		expected := before.Add(30 * time.Minute)
		ts := time.Unix(int64(result), 0)
		assert.WithinDuration(t, expected, ts, timeTolerance)
	})

	t.Run("now+2days", func(t *testing.T) {
		before := time.Now()
		result, err := parseBeginTime("now+2days")
		require.NoError(t, err)
		expected := before.Add(2 * 24 * time.Hour)
		ts := time.Unix(int64(result), 0)
		assert.WithinDuration(t, expected, ts, timeTolerance)
	})

	t.Run("now+1week", func(t *testing.T) {
		before := time.Now()
		result, err := parseBeginTime("now+1week")
		require.NoError(t, err)
		expected := before.Add(7 * 24 * time.Hour)
		ts := time.Unix(int64(result), 0)
		assert.WithinDuration(t, expected, ts, timeTolerance)
	})
}

func TestParseBeginTime_ISODates(t *testing.T) {
	t.Run("date only", func(t *testing.T) {
		result, err := parseBeginTime("2024-06-15")
		require.NoError(t, err)
		expected, _ := time.Parse("2006-01-02", "2024-06-15")
		assert.Equal(t, uint64(expected.Unix()), result)
	})

	t.Run("date and time without seconds", func(t *testing.T) {
		result, err := parseBeginTime("2024-06-15T14:30")
		require.NoError(t, err)
		expected, _ := time.Parse("2006-01-02T15:04", "2024-06-15T14:30")
		assert.Equal(t, uint64(expected.Unix()), result)
	})

	t.Run("date and time with seconds", func(t *testing.T) {
		result, err := parseBeginTime("2024-06-15T14:30:00")
		require.NoError(t, err)
		expected, _ := time.Parse("2006-01-02T15:04:05", "2024-06-15T14:30:00")
		assert.Equal(t, uint64(expected.Unix()), result)
	})
}

func TestParseBeginTime_USDates(t *testing.T) {
	t.Run("MM/DD/YY", func(t *testing.T) {
		result, err := parseBeginTime("06/15/24")
		require.NoError(t, err)
		expected, _ := time.Parse("01/02/06", "06/15/24")
		assert.Equal(t, uint64(expected.Unix()), result)
	})

	t.Run("MMDDYY", func(t *testing.T) {
		result, err := parseBeginTime("061524")
		require.NoError(t, err)
		expected, _ := time.Parse("010206", "061524")
		assert.Equal(t, uint64(expected.Unix()), result)
	})
}

func TestParseBeginTime_TimeOfDay(t *testing.T) {
	t.Run("HH:MM", func(t *testing.T) {
		result, err := parseBeginTime("16:00")
		require.NoError(t, err)

		now := time.Now()
		target := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())
		if now.After(target) {
			target = target.Add(24 * time.Hour)
		}
		assert.Equal(t, uint64(target.Unix()), result)
	})
}

func TestParseBeginTime_AMPM(t *testing.T) {
	t.Run("4:00PM", func(t *testing.T) {
		result, err := parseBeginTime("4:00PM")
		require.NoError(t, err)

		now := time.Now()
		target := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())
		if now.After(target) {
			target = target.Add(24 * time.Hour)
		}
		assert.Equal(t, uint64(target.Unix()), result)
	})

	t.Run("9AM", func(t *testing.T) {
		result, err := parseBeginTime("9AM")
		require.NoError(t, err)

		now := time.Now()
		target := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
		if now.After(target) {
			target = target.Add(24 * time.Hour)
		}
		assert.Equal(t, uint64(target.Unix()), result)
	})
}

func TestParseBeginTime_RFC3339(t *testing.T) {
	result, err := parseBeginTime("2024-06-15T14:30:00Z")
	require.NoError(t, err)
	expected, _ := time.Parse(time.RFC3339, "2024-06-15T14:30:00Z")
	assert.Equal(t, uint64(expected.Unix()), result)
}

func TestParseBeginTime_Invalid(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		_, err := parseBeginTime("")
		assert.Error(t, err)
	})

	t.Run("garbage", func(t *testing.T) {
		_, err := parseBeginTime("garbage")
		assert.Error(t, err)
	})

	t.Run("whitespace only", func(t *testing.T) {
		_, err := parseBeginTime("   ")
		assert.Error(t, err)
	})
}

func TestParseDurationUnit(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		unit     string
		expected time.Duration
	}{
		{"seconds short", 30, "s", 30 * time.Second},
		{"seconds word", 30, "seconds", 30 * time.Second},
		{"seconds singular", 1, "second", 1 * time.Second},
		{"sec", 10, "sec", 10 * time.Second},
		{"minutes short", 5, "m", 5 * time.Minute},
		{"minutes word", 5, "minutes", 5 * time.Minute},
		{"minutes singular", 1, "minute", 1 * time.Minute},
		{"min", 10, "min", 10 * time.Minute},
		{"hours short", 2, "h", 2 * time.Hour},
		{"hours word", 2, "hours", 2 * time.Hour},
		{"hour singular", 1, "hour", 1 * time.Hour},
		{"days short", 3, "d", 3 * 24 * time.Hour},
		{"days word", 3, "days", 3 * 24 * time.Hour},
		{"day singular", 1, "day", 1 * 24 * time.Hour},
		{"weeks short", 1, "w", 7 * 24 * time.Hour},
		{"weeks word", 2, "weeks", 2 * 7 * 24 * time.Hour},
		{"week singular", 1, "week", 7 * 24 * time.Hour},
		{"unknown unit", 5, "fortnights", 0},
		{"empty unit", 5, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDurationUnit(tt.n, tt.unit)
			assert.Equal(t, tt.expected, result)
		})
	}
}
