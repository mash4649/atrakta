package util

import (
	"fmt"
	"sync/atomic"
	"time"
)

var seq atomic.Uint64

func NewEventID() string {
	n := seq.Add(1)
	return fmt.Sprintf("%d-%06d", time.Now().UnixNano(), n)
}

func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
