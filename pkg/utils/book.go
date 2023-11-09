package utils

import (
	"sync"
	"time"

	"github.com/migalabs/goteth/pkg/metrics"
)

var (
	emptyKey   = ""
	structName = "routinebook"
)

type RoutineBook struct {
	sync.Mutex
	pages         map[string]string
	freeSpaceChan chan struct{}
	size          int64
	bookTag       string
}

func NewRoutineBook(size int, tag string) *RoutineBook {

	r := &RoutineBook{
		pages:         make(map[string]string, size), // contains a list of keys identifying routines
		freeSpaceChan: make(chan struct{}, size),     // indicates the free position in the array
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
	select {
	case <-ticker.C:
		log.WithField("bookTag", r.bookTag).Warnf("Waiting for too long to acquire page %s...", key)
	case <-r.freeSpaceChan:
		r.Set(key, "active")
	}
}

func (r *RoutineBook) FreePage(key string) {

	r.Lock()
	defer r.Unlock()
	_, ok := r.pages[key]
	// If the key exists
	if ok {
		delete(r.pages, key)
		r.freeSpaceChan <- struct{}{}
	}

}

func (r *RoutineBook) CheckPageActive(key string) bool {
	r.Lock()

	_, ok := r.pages[key]

	r.Unlock()
	return ok

}

func (r *RoutineBook) Set(key string, value string) {
	r.Lock()
	defer r.Unlock()
	r.pages[key] = value // book page

}

func (r *RoutineBook) ActivePages() int {
	r.Lock()
	defer r.Unlock()
	result := 0
	for _, item := range r.pages {
		if item != emptyKey {
			result += 1
		}
	}

	return result
}

func (r *RoutineBook) NumFreePages() int {

	r.Lock()
	defer r.Unlock()
	return int(r.size) - len(r.pages)
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
