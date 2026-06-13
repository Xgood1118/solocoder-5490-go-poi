package utils

import (
	"poi-service/internal/model"
	"strconv"
	"time"
)

type BusinessStatus string

const (
	StatusOpen       BusinessStatus = "营业中"
	StatusClosed     BusinessStatus = "已打烊"
	StatusSoonClose  BusinessStatus = "即将打烊"
)

func GetBusinessStatus(bh model.BusinessHours, now time.Time) BusinessStatus {
	weekday := now.Weekday()
	periods := getPeriodsForWeekday(bh, weekday)

	current := now.Format("15:04")

	for _, p := range periods {
		if isTimeInPeriod(current, p.Open, p.Close) {
			closeMin := parseTime(p.Close)
			currMin := parseTime(current)
			if closeMin-currMin <= 30 && closeMin > currMin {
				return StatusSoonClose
			}
			return StatusOpen
		}
	}
	return StatusClosed
}

func getPeriodsForWeekday(bh model.BusinessHours, weekday time.Weekday) []model.BusinessHoursPeriod {
	switch weekday {
	case time.Monday:
		return bh.Monday
	case time.Tuesday:
		return bh.Tuesday
	case time.Wednesday:
		return bh.Wednesday
	case time.Thursday:
		return bh.Thursday
	case time.Friday:
		return bh.Friday
	case time.Saturday:
		return bh.Saturday
	case time.Sunday:
		return bh.Sunday
	}
	return nil
}

func isTimeInPeriod(current, open, close string) bool {
	currMin := parseTime(current)
	openMin := parseTime(open)
	closeMin := parseTime(close)

	if closeMin > openMin {
		return currMin >= openMin && currMin < closeMin
	}
	return currMin >= openMin || currMin < closeMin
}

func parseTime(t string) int {
	if len(t) != 5 {
		return 0
	}
	hour, _ := strconv.Atoi(t[:2])
	minute, _ := strconv.Atoi(t[3:])
	return hour*60 + minute
}
