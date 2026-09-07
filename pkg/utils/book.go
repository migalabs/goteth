package utils

import (
	"sync"
	"time"

	"github.com/migalabs/goteth/pkg/metrics"
)

var (
	structName        = "routinebook"
	CheckPageInterval = 1 * time.Second
)

// RoutineBook is a counting semaphore that also records what is holding each
// slot, so callers can ask whether a given key is still being worked on.
//
// pages counts holders rather than marking presence. Acquire is not a per-key
// lock: two goroutines can acquire the same key, and each takes its own token.
// While a page was a bare marker the second acquire overwrote the first and
// only one FreePage found a key to delete, so one token was never returned and
// the pool shrank by one every time it happened (migalabs/goteth#292).
type RoutineBook struct {
	sync.Mutex
	pages         map[string]int
	freeSpaceChan chan struct{}
	size          int64
	bookTag       string
}

func NewRoutineBook(size int, tag string) *RoutineBook {

	r := &RoutineBook{
		pages:         make(map[string]int, size), // holders per key, see the type comment
		freeSpaceChan: make(chan struct{}, size),  // indicates the free position in the array
		size:          int64(size),
		bookTag:       tag,
	}
	r.Init()
	return r

}

func (r *RoutineBook) Init() {
	for i := 0; i < int(r.size); i++ {
		r.freeSpaceChan <- struct{}{}
	}
}

func (r *RoutineBook) Acquire(key string) {

	ticker := time.NewTicker(AcquireWaitIntervalLog)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.WithField("bookTag", r.bookTag).Warnf("Waiting for too long to acquire page %s...", key)
		case <-r.freeSpaceChan:
			r.hold(key)
			return
		}
	}
}

// FreePage releases one holder of key and returns its token.
//
// The token goes back for every held acquire, not once per key: releasing only
// once would strand the tokens of any other holder of the same key. A key with
// no holders returns nothing, which keeps the channel from being filled past
// its capacity - a send that blocked there would never complete.
func (r *RoutineBook) FreePage(key string) {

	r.Lock()
	holders, ok := r.pages[key]
	if !ok || holders <= 0 {
		r.Unlock()
		return
	}
	if holders == 1 {
		delete(r.pages, key)
	} else {
		r.pages[key] = holders - 1
	}
	r.Unlock()

	// Outside the lock: a send here can only block if the channel is full,
	// and blocking while holding the mutex would stop every other caller.
	r.freeSpaceChan <- struct{}{}
}

func (r *RoutineBook) CheckPageActive(key string) bool {

	_, ok := r.get(key)

	return ok

}

func (r *RoutineBook) WaitUntilInactive(key string) bool {
	ticker := time.NewTicker(CheckPageInterval)

	for range ticker.C {

		_, ok := r.get(key)

		if !ok {
			return true
		}
	}

	return false

}

// hold records one more holder of key. The caller has already taken a token.
func (r *RoutineBook) hold(key string) {
	r.Lock()
	defer r.Unlock()
	r.pages[key]++
}

func (r *RoutineBook) get(key string) (int, bool) {
	r.Lock()
	defer r.Unlock()

	result, ok := r.pages[key]

	return result, ok

}

// ActivePages is the number of holders across all keys, which is what the
// shutdown checks mean by "anything still running". Two holders of one key
// count twice, because two routines are running.
func (r *RoutineBook) ActivePages() int {
	r.Lock()
	defer r.Unlock()
	result := 0
	for _, holders := range r.pages {
		result += holders
	}

	return result
}

// NumFreePages counts the tokens still available. It reads the channel rather
// than subtracting the number of keys: a key can have several holders, so the
// key count stopped matching the tokens in use once pages began counting.
func (r *RoutineBook) NumFreePages() int {
	return len(r.freeSpaceChan)
}

func (r *RoutineBook) GetKeys() []string {
	r.Lock()
	defer r.Unlock()
	keys := make([]string, 0, len(r.pages))
	for k := range r.pages {
		keys = append(keys, k)
	}
	return keys
}

func (r *RoutineBook) GetPrometheusMetrics() *metrics.MetricsModule {
	metricsMod := metrics.NewMetricsModule(
		structName,
		r.bookTag,
	)
	// compose all the metrics
	metricsMod.AddIndvMetric(r.getCurrentKeys())

	return metricsMod
}

func (r *RoutineBook) getCurrentKeys() *metrics.IndvMetrics {
	initFn := func() error {
		return nil
	}
	updateFn := func() (interface{}, error) {
		keyList := r.GetKeys()
		return keyList, nil
	}
	currentKeys, err := metrics.NewIndvMetrics(
		r.bookTag+"-current_keys",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return currentKeys
}
