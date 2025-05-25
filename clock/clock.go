package clock

import (
	"fmt"
	"time"
)

type Clock struct {
	Hour   int
	Minute int
	Second int
	Time   time.Time
}

func NewClock(t time.Time) *Clock {
	h, m, s := t.Hour(), t.Minute(), t.Second()
	fmt.Printf("new clock; %s\n", t)
	return &Clock{
		Hour:   h,
		Minute: m,
		Second: s,
		Time:   t.Truncate(0),
	}
}
