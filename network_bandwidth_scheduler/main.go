// Package network_bandwidth_scheduler implements a service that applies bandwidth
// limitations during certain time windows. The default behavior is to limit bandwidth at
// all times, unless different values of "start" and "end" time are given.
package main

import (
	"context"
	"flag"
	"log"

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

	s, err := scheduler.New(*nic, *start, *end, &real.Clock{})
	if err != nil {
		log.Fatalf("Cannot initialize scheduler: %v", err)
	}
	s.Run(context.Background())
}
