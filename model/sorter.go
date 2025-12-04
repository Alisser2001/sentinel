package model

import (
	"sort"
	"strings"
)

type SortColumn int

const (
	SortByCPUCol SortColumn = iota
	SortByMEM
	SortByPID
	SortByUSER
	SortByVSIZE
	SortByRSS
	SortByTIME
)

type Sorter struct {
	Column     SortColumn
	Descending bool
}

func NewSorter() *Sorter {
	return &Sorter{
		Column:     SortByCPUCol,
		Descending: true, // Default: highest CPU first
	}
}

func (s *Sorter) Toggle(col SortColumn) {
	if s.Column == col {
		s.Descending = !s.Descending
	} else {
		s.Column = col
		s.Descending = true
	}
}

func (s *Sorter) Sort(records []ProcRec) {
	sort.Slice(records, func(i, j int) bool {
		a, b := &records[i], &records[j]

		var less bool
		switch s.Column {
		case SortByCPUCol:
			less = a.CPU < b.CPU
		case SortByMEM:
			less = a.PMem < b.PMem
		case SortByPID:
			less = a.Pid < b.Pid
		case SortByUSER:
			less = strings.ToLower(a.User) < strings.ToLower(b.User)
		case SortByVSIZE:
			less = a.VSizeKB < b.VSizeKB
		case SortByRSS:
			less = a.RSSKB < b.RSSKB
		case SortByTIME:
			less = a.CurProcTime < b.CurProcTime
		default:
			less = a.CPU < b.CPU
		}

		if s.Descending {
			return !less
		}
		return less
	})
}

func (s *Sorter) ColumnName() string {
	names := []string{"CPU", "MEM", "PID", "USER", "VSIZE", "RSS", "TIME"}
	return names[s.Column]
}
