package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/karagog/clock-go/simulated"
)

type throttlingObserver struct {
	origApplyThrottling  func(string)
	origClearThrottling  func(string)
	applyThrottlingCount int
	clearThrottlingCount int
}

func newThrottlingObserver() *throttlingObserver {
	o := &throttlingObserver{
		origApplyThrottling: applyThrottling,
		origClearThrottling: clearThrottling,
	}
	applyThrottling = o.applyThrottling
	clearThrottling = o.clearThrottling
	return o
}

func (o *throttlingObserver) applyThrottling(string) {
	o.applyThrottlingCount++
}

func (o *throttlingObserver) clearThrottling(string) {
	o.clearThrottlingCount++
}

func (o *throttlingObserver) close() {
	applyThrottling = o.origApplyThrottling
	clearThrottling = o.origClearThrottling
}

func TestComputeNextOccurrence(t *testing.T) {
	testCases := []struct {
		desc      string
		now       time.Time
		timeOfDay time.Duration
		wantTime  time.Time
	}{
		{
			desc:      "nominal",
			now:       time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC),
			timeOfDay: 2 * time.Hour, // 2am
			wantTime:  time.Date(2023, 12, 30, 2, 0, 0, 0, time.UTC),
		},
		{
			desc:      "same time as now",
			now:       time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC),
			timeOfDay: 0 * time.Hour, // 12am
			wantTime:  time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			desc:      "time before now",
			now:       time.Date(2023, 12, 30, 3, 0, 0, 0, time.UTC),
			timeOfDay: 1 * time.Hour, // 1am
			wantTime:  time.Date(2023, 12, 31, 1, 0, 0, 0, time.UTC),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			clk := simulated.NewClock(tc.now)
			nextTime := computeNextOccurrence(tc.timeOfDay, clk)
			if nextTime != tc.wantTime {
				t.Fatalf("computeNextOccurrence(%v, %v) = %v, want %v", tc.timeOfDay, tc.now, nextTime, tc.wantTime)
			}
		})
	}
}

func TestNewStartThrottling(t *testing.T) {
	o := newThrottlingObserver()
	defer o.close()
	clk := simulated.NewClock(time.Date(2023, 12, 30, 1, 0, 0, 0, time.UTC))
	s, err := New("foo", 0, 12*time.Hour, clk)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("Got nil, want a scheduler")
	}
	if o.applyThrottlingCount > 0 {
		t.Fatalf("Applied throttling %v times, want 0", o.applyThrottlingCount)
	}
	if o.clearThrottlingCount > 0 {
		t.Fatalf("Cleared throttling %v times, want 0", o.clearThrottlingCount)
	}

	// Make sure we can run it, and that it obeys the context cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(ctx)
	}()
	cancel()
	<-done

	if o.applyThrottlingCount != 1 {
		t.Fatalf("Applied throttling %v times, want 1", o.applyThrottlingCount)
	}
	if o.clearThrottlingCount > 0 {
		t.Fatalf("Cleared throttling %v times, want 0", o.clearThrottlingCount)
	}
}

func TestNewStartClearThrottling(t *testing.T) {
	o := newThrottlingObserver()
	defer o.close()
	clk := simulated.NewClock(time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC))
	s, err := New("foo", 1, 12*time.Hour, clk)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(ctx)
	}()
	cancel()
	<-done

	if o.applyThrottlingCount != 0 {
		t.Fatalf("Applied throttling %v times, want 0", o.applyThrottlingCount)
	}
	if o.clearThrottlingCount != 1 {
		t.Fatalf("Cleared throttling %v times, want 1", o.clearThrottlingCount)
	}
}

func TestNewAlwaysThrottling(t *testing.T) {
	o := newThrottlingObserver()
	defer o.close()
	clk := simulated.NewClock(time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC))
	s, err := New("foo", 0, 0, clk)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(ctx)
	}()
	clk.Advance(1000 * time.Hour)
	time.Sleep(40 * time.Millisecond)
	cancel()
	<-done

	if o.applyThrottlingCount != 1 {
		t.Fatalf("Applied throttling %v times, want 1", o.applyThrottlingCount)
	}
	if o.clearThrottlingCount != 0 {
		t.Fatalf("Cleared throttling %v times, want 0", o.clearThrottlingCount)
	}
}

func TestNewNegativeStartTimeError(t *testing.T) {
	if _, err := New("foo", -time.Hour, 0, simulated.NewClock(time.Now())); err == nil {
		t.Fatal("Got nil error, want error")
	}
}

func TestNewNegativeEndTimeError(t *testing.T) {
	if _, err := New("foo", 0, -time.Hour, simulated.NewClock(time.Now())); err == nil {
		t.Fatal("Got nil error, want error")
	}
}

func TestLongRunningService(t *testing.T) {
	o := newThrottlingObserver()
	defer o.close()
	clk := simulated.NewClock(time.Date(2023, 7, 1, 1, 0, 0, 0, time.UTC))
	s, err := New("foo", 0, 12*time.Hour, clk)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(ctx)
	}()
	time.Sleep(40 * time.Millisecond)
	clk.Advance(12 * time.Hour)
	time.Sleep(40 * time.Millisecond)
	clk.Advance(12 * time.Hour)
	time.Sleep(40 * time.Millisecond)
	cancel()
	<-done

	if o.applyThrottlingCount != 2 {
		t.Fatalf("Applied throttling %v times, want 1", o.applyThrottlingCount)
	}
	if o.clearThrottlingCount != 1 {
		t.Fatalf("Cleared throttling %v times, want 1", o.clearThrottlingCount)
	}
}
