package commander

import "time"

type AttackOrders struct {
	targetUrl          string
	timeBetweenAttacks time.Duration
}
