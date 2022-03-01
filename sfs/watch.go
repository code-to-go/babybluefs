package sfs

import "time"

func Watch(mesh *Mesh, name string, period time.Duration, mon chan string) (*time.Ticker) {
	ticker := time.NewTicker(period)

	go func() {
		for {
			last := time.Time{}
			select {
			case <- ticker.C:
				if mesh.Zombie {
					ticker.Stop()
					close(mon)
					return
				}
				Sync(mesh, name, last, mon)
				last = time.Now()
			}
		}
	}()
	return ticker
}
