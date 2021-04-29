package datahistoryjobresult

import "time"

type DataHistoryJobResult struct {
	ID                string
	JobID             string
	IntervalStartDate time.Time
	IntervalEndDate   time.Time
	Status            int64
	Error             string
	Date              time.Time
}
