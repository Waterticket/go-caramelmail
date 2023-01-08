package caramelmail

import (
	"github.com/sony/gobreaker"
	"time"
)

func CircuitBreaker(domain string) *gobreaker.CircuitBreaker {
	if circuitBreakerList[domain] == nil {
		circuitBreakerList[domain] = gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "mail:" + domain,
			MaxRequests: 3,
			Interval:    60 * time.Second,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return counts.Requests >= 3 && failureRatio >= 0.6
			},
		})
	}

	return circuitBreakerList[domain]
}
