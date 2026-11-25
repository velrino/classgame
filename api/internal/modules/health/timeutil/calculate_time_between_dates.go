package timeutil

import "time"

type TimeBetweenDatesDTO struct {
	Start   string `json:"start"`
	End     string `json:"end"`
	Years   int    `json:"years"`
	Months  int    `json:"months"`
	Days    int    `json:"days"`
	Hours   int    `json:"hours"`
	Minutes int    `json:"minutes"`
	Seconds int    `json:"seconds"`
}

func CalculateTimeBetweenDates(start time.Time, end time.Time) TimeBetweenDatesDTO {
	if end.Before(start) {
		start, end = end, start
	}

	years, months, cursor := calculateYearsAndMonths(start, end)
	duration := end.Sub(cursor)

	days := int(duration / (24 * time.Hour))
	duration -= time.Duration(days) * 24 * time.Hour

	hours := int(duration / time.Hour)
	duration -= time.Duration(hours) * time.Hour

	minutes := int(duration / time.Minute)
	duration -= time.Duration(minutes) * time.Minute

	seconds := int(duration / time.Second)

	return TimeBetweenDatesDTO{
		Start:   start.Format(time.RFC3339Nano),
		End:     end.Format(time.RFC3339Nano),
		Years:   years,
		Months:  months,
		Days:    days,
		Hours:   hours,
		Minutes: minutes,
		Seconds: seconds,
	}
}

func calculateYearsAndMonths(start time.Time, end time.Time) (int, int, time.Time) {
	years := 0
	cursor := start

	for {
		next := addDateClamped(cursor, 1, 0)
		if next.After(end) {
			break
		}

		years++
		cursor = next
	}

	months := 0

	for {
		next := addDateClamped(cursor, 0, 1)
		if next.After(end) {
			break
		}

		months++
		cursor = next
	}

	return years, months, cursor
}

func addDateClamped(value time.Time, years int, months int) time.Time {
	targetYear := value.Year() + years
	targetMonth := int(value.Month()) + months

	for targetMonth > 12 {
		targetYear++
		targetMonth -= 12
	}
	for targetMonth < 1 {
		targetYear--
		targetMonth += 12
	}

	month := time.Month(targetMonth)
	day := value.Day()
	lastDay := daysInMonth(targetYear, month, value.Location())
	if day > lastDay {
		day = lastDay
	}

	return time.Date(
		targetYear,
		month,
		day,
		value.Hour(),
		value.Minute(),
		value.Second(),
		value.Nanosecond(),
		value.Location(),
	)
}

func daysInMonth(year int, month time.Month, location *time.Location) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, location).Day()
}
