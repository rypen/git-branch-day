package timeutil

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseClock(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, 0, errors.New("time must be HH:MM")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, errors.New("time must be HH:MM")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, errors.New("time must be HH:MM")
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, errors.New("time must be HH:MM")
	}
	return hour, minute, nil
}

func TimeRangeForDay(day time.Time, startValue, endValue string) (time.Time, time.Time, error) {
	hourStart, minStart, err := ParseClock(startValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	hourEnd, minEnd, err := ParseClock(endValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := time.Date(day.Year(), day.Month(), day.Day(), hourStart, minStart, 0, 0, day.Location())
	end := time.Date(day.Year(), day.Month(), day.Day(), hourEnd, minEnd, 0, 0, day.Location())
	if !end.After(start) {
		return time.Time{}, time.Time{}, errors.New("end time must be after start time")
	}
	return start, end, nil
}

func AllocateTimes(start, end time.Time, efforts []int) ([]time.Time, error) {
	if !end.After(start) {
		return nil, errors.New("end time must be after start time")
	}
	if len(efforts) == 0 {
		return nil, errors.New("no efforts provided")
	}

	totalEffort := 0
	for _, effort := range efforts {
		if effort < 0 {
			return nil, fmt.Errorf("invalid effort %d", effort)
		}
		totalEffort += effort
	}
	if totalEffort == 0 {
		totalEffort = len(efforts)
		efforts = make([]int, len(efforts))
		for i := range efforts {
			efforts[i] = 1
		}
	}

	totalSeconds := int64(end.Sub(start).Seconds())
	if totalSeconds <= 0 {
		return nil, errors.New("invalid time range")
	}

	times := make([]time.Time, len(efforts))
	remaining := totalSeconds
	elapsed := int64(0)
	for i, effort := range efforts {
		var seconds int64
		if i == len(efforts)-1 {
			seconds = remaining
		} else {
			seconds = int64(float64(totalSeconds) * float64(effort) / float64(totalEffort))
			if seconds < 0 {
				seconds = 0
			}
			if seconds > remaining {
				seconds = remaining
			}
		}
		elapsed += seconds
		remaining = totalSeconds - elapsed
		times[i] = start.Add(time.Duration(elapsed) * time.Second)
	}

	return times, nil
}
