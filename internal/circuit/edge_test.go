package circuit

import (
	"sync"
	"testing"
	"time"
)

func TestHalfOpen_SuccessCloses(t *testing.T) {
	b := New(2, 20*time.Millisecond)
	b.Record(500)
	b.Record(500) // opens
	if b.Allow() {
		t.Fatal("should be open")
	}
	time.Sleep(25 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("should be half-open after cooldown")
	}
	b.Record(200) // probe succeeds
	if !b.Allow() {
		t.Fatal("should be closed after successful probe")
	}
}

func TestHalfOpen_FailureReopens(t *testing.T) {
	b := New(2, 20*time.Millisecond)
	b.Record(500)
	b.Record(500) // opens
	time.Sleep(25 * time.Millisecond)
	b.Allow()     // half-open probe
	b.Record(500) // probe fails — should reopen
	if b.Allow() {
		t.Fatal("should be re-opened after failed probe")
	}
}

func TestHalfOpen_OnlyOneProbe(t *testing.T) {
	b := New(2, 20*time.Millisecond)
	b.Record(500)
	b.Record(500)
	time.Sleep(25 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("first probe should be allowed")
	}
	// Second concurrent request should be rejected (half-open allows only one)
	if b.Allow() {
		t.Fatal("second request during half-open should be rejected")
	}
}

func TestRetryAfter_DecreasesOverTime(t *testing.T) {
	b := New(1, 100*time.Millisecond)
	b.Record(500) // opens
	ra1 := b.RetryAfter()
	time.Sleep(50 * time.Millisecond)
	ra2 := b.RetryAfter()
	if ra2 > ra1 {
		t.Fatalf("retry-after should decrease: %d then %d", ra1, ra2)
	}
}

func TestConcurrent_NoRace(t *testing.T) {
	b := New(100, time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b.Allow()
			if i%3 == 0 {
				b.Record(500)
			} else {
				b.Record(200)
			}
			b.RetryAfter()
			b.State()
		}(i)
	}
	wg.Wait()
}

func TestZeroThreshold_OpensImmediately(t *testing.T) {
	b := New(0, 20*time.Millisecond)
	b.Record(500)
	// threshold 0 means any failure opens the circuit
	if b.Allow() {
		t.Fatal("zero threshold should open on first failure")
	}
}
