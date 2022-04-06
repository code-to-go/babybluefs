package mesh

import (
	"time"
)

func Watch(m *Mesh, name string, period time.Duration, mon chan string) *time.Ticker {
	ticker := time.NewTicker(period)

	go func() {
		for {
			last := time.Time{}
			select {
			case <-ticker.C:
				if m.Zombie {
					ticker.Stop()
					close(mon)
					return
				}
				Sync(m, name, last, mon)
				last = time.Now()
			}
		}
	}()
	return ticker
}
