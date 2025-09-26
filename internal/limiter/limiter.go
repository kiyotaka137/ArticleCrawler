package limiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type DomainLimiter struct {
	m          sync.Map // здесь хранится [string]*rate.Limiter
	defaultRPS int
	burst      int
}

func NewDomainLimiter(defaultRPS, burst int) *DomainLimiter {
	if defaultRPS <= 0 {
		defaultRPS = 2
	}
	if burst <= 0 {
		burst = 5
	}
	return &DomainLimiter{
		defaultRPS: defaultRPS,
		burst:      burst,
	}
}

func (d *DomainLimiter) getLimiter(domain string) *rate.Limiter {
	if v, ok := d.m.Load(domain); ok {
		return v.(*rate.Limiter)
	}
	l := rate.NewLimiter(rate.Limit(d.defaultRPS), d.burst)
	actual, _ := d.m.LoadOrStore(domain, l)
	return actual.(*rate.Limiter)
}

func (d *DomainLimiter) Allow(domain string) bool {
	l := d.getLimiter(domain)
	return l.Allow()
}

func (d *DomainLimiter) ReserveN(domain string, n int) *rate.Reservation {
	return d.getLimiter(domain).ReserveN(time.Now(), n)
}
