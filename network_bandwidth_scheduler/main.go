// Package network_bandwidth_scheduler implements a service that applies bandwidth
// limitations during certain time windows. The default behavior is to limit bandwidth at
// all times, unless different values of "start" and "end" time are given.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"example.com/sysadmin/network_bandwidth_scheduler/scheduler"
	"github.com/karagog/clock-go/real"
)

var start = flag.Duration("throttle_start_time", 0, "Throttling starts at this time of day.")
var end = flag.Duration("throttle_end_time", 0, "Throttling ends at this time of day.")
var nic = flag.String("nic", "", "The network interface to apply throttling.")

func main() {
	flag.Parse()
	if *nic == "" {
		log.Fatal("--nic must be specified")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle termination signals so we can exit gracefully.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case s := <-ch:
			log.Printf("Signal received (%v). Exiting...", s)
			cancel()
		case <-ctx.Done():
		}
	}()

	// Run the scheduler until canceled.
	s, err := scheduler.New(*nic, *start, *end, &real.Clock{})
	if err != nil {
		log.Fatalf("Cannot initialize scheduler: %v", err)
	}
	defer s.Close()
	s.Run(ctx)
}
