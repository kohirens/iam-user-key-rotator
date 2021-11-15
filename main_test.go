package main

import (
	"testing"
	"time"
)

func TestDaysOld(tester *testing.T) {
	cases := []struct{
		name string
		fix time.Time
	}{
		{"10days", time.Now().AddDate(0, 0, -10) },
	}

	for _, test := range cases {
		tester.Run(test.name, func(t *testing.T) {
			got := DaysOld(&test.fix)
			if got != 10 {
				t.Errorf("want %v, got %v", test.fix, got)
			}
		})
	}
}