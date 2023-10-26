package utils

import (
	"sync"
	"time"
)

var (
	emptyKey = ""
)

type RoutineBook struct {
	sync.Mutex
	pages         map[string]string
	freeSpaceChan chan struct{}
	size          int64
}

func NewRoutineBook(size int) *RoutineBook {

	r := &RoutineBook{
		pages:         make(map[string]string, size), // contains a list of keys identifying routines
		freeSpaceChan: make(chan struct{}, size),     // indicates the free position in the array
		size:          int64(size),
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

	ticker := time.NewTicker(WaitMaxTimeout)
	select {
	case <-ticker.C:
		log.Fatalf("Waiting for too long to acquire page %s...", key)
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
