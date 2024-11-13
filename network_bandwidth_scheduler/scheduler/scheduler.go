package scheduler

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/karagog/clock-go"
)

var wondershaperPath = flag.String("wondershaper_path", "/usr/local/sbin/wondershaper", "The path to 'wondershaper'")
var downloadKbps = flag.Int("download_kbps", 10000, "Throttle the download rate to this value.")
var uploadKbps = flag.Int("upload_kbps", 10000, "Throttle the upload rate to this value.")

var applyThrottling = func(nic string) {
	cmd := exec.Command(*wondershaperPath, "-a", nic, "-d", fmt.Sprintf("%d", *downloadKbps), "-u", fmt.Sprintf("%d", *uploadKbps))
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing wondershaper: %s\n", err)
		return
	}
	log.Printf("Now throttling interface '%s' to %d Kbps download and %d Kbps upload\n", nic, *downloadKbps, *uploadKbps)
}

var clearThrottling = func(nic string) {
	cmd := exec.Command(*wondershaperPath, "-c", "-a", nic)
	// Ignore the error, because the latest version of wondershaper always returns non-zero even if this was successful.
	cmd.Run()
	log.Printf("Removed throttling on interface '%s'\n", nic)
}

// This assumes the given duration is greater than 0 and less than 24 hours.
func computeNextOccurrence(timeOfDay time.Duration, clk clock.Clock) time.Time {
	now := clk.Now()
	t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(timeOfDay)
	if t.Before(now) {
		t = t.Add(24 * time.Hour)
	}
	return t
}

type Scheduler struct {
	nic                                        string
	clock                                      clock.Clock
	nextThrottleStartTime, nextThrottleEndTime time.Time
	nextUpdateTime                             time.Time
	throttling                                 bool
}

func New(nic string, throttleTimeStart, throttleTimeEnd time.Duration, clock clock.Clock) (*Scheduler, error) {
	if throttleTimeStart < 0 || throttleTimeEnd < 0 ||
		throttleTimeStart >= 24*time.Hour || throttleTimeEnd >= 24*time.Hour {
		return nil, fmt.Errorf("times must be greater than 0 and less than 24 hours, got: %v, %v", throttleTimeStart, throttleTimeEnd)
	}

	// Clear any previous throttling that may be on this nic. If we don't, then the new
	// settings won't be applied.
	clearThrottling(nic)

	return &Scheduler{
		nic:                   nic,
		clock:                 clock,
		nextThrottleStartTime: computeNextOccurrence(throttleTimeStart, clock),
		nextThrottleEndTime:   computeNextOccurrence(throttleTimeEnd, clock),
	}, nil
}

func (s *Scheduler) Run(ctx context.Context) {
	// Update throttling for the first time on startup.
	if s.nextThrottleStartTime == s.nextThrottleEndTime {
		applyThrottling(s.nic)
		return // exit gracefully since we want to throttle ad infinitum
	} else if s.nextThrottleEndTime.Before(s.nextThrottleStartTime) {
		applyThrottling(s.nic)
		s.nextUpdateTime = s.nextThrottleEndTime
		s.throttling = true
	} else {
		clearThrottling(s.nic)
		s.nextUpdateTime = s.nextThrottleStartTime
		s.throttling = false
	}

	// Loop forever and update the throttling when necessary.
	for {
		now := s.clock.Now()
		t := s.clock.NewTimer(s.nextUpdateTime.Sub(now))
		select {
		case <-t.C():
			t.Stop()
			s.toggleBandwidthEnforcement()
		case <-ctx.Done():
			// Clear throttling on service exit.
			clearThrottling(s.nic)
			return
		}
	}
}

func (s *Scheduler) toggleBandwidthEnforcement() {
	if s.throttling {
		clearThrottling(s.nic)
		s.throttling = false
		s.nextThrottleEndTime = s.nextThrottleEndTime.Add(24 * time.Hour)
		s.nextUpdateTime = s.nextThrottleStartTime
	} else {
		applyThrottling(s.nic)
		s.throttling = true
		s.nextThrottleStartTime = s.nextThrottleStartTime.Add(24 * time.Hour)
		s.nextUpdateTime = s.nextThrottleEndTime
	}
}
