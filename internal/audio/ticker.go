package audio

import "time"

// pollInterval is the interval at which the StreamCapturer polls for new samples.
const pollInterval = 16 * time.Millisecond // ~60Hz, well below VAD window rate at 16kHz

// tickerIface abstracts time.Ticker for testing.
type tickerIface interface {
	C() <-chan time.Time
	Stop()
}

type realTicker struct {
	t *time.Ticker
}

func (r *realTicker) C() <-chan time.Time { return r.t.C }
func (r *realTicker) Stop()               { r.t.Stop() }

// makeTickerFunc is a variable so tests can override the ticker.
var makeTickerFunc = func() tickerIface {
	return &realTicker{t: time.NewTicker(pollInterval)}
}
