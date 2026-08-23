package spotify

import (
	"sync"
	"testing"
)

func TestFetchManagerConcurrentBeginCancelNoRace(t *testing.T) {
	fm := NewFetchManager()

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ctx, tok := fm.Begin()
			_ = ctx
			fm.Clear(tok)
		}()
		go func() {
			defer wg.Done()
			fm.Cancel()
		}()
	}
	wg.Wait()
}
