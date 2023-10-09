package utils

import (
	"sync"
)

var (
	emptyKey = ""
	nullPos  = -1
)

type RoutineBook struct {
	sync.Mutex
	arr           []string
	freeSpaceChan chan int
	countPages    int64
}

func NewRoutineBook(size int) *RoutineBook {

	r := &RoutineBook{
		arr:           make([]string, size), // contains a list of keys identifying routines
		freeSpaceChan: make(chan int, size), // indicates the free position in the array
	}
	r.Init()
	return r

}

func (r *RoutineBook) Init() {
	for i := 0; i < len(r.arr); i++ {
		r.freeSpaceChan <- i
	}
}

func (r *RoutineBook) searchFreePage() int {
	return r.searchPage(emptyKey)
}

func (r *RoutineBook) searchPage(key string) int {
	for page, routine := range r.arr {
		if routine == key {
			// there is space for the routine
			return page
		}
	}
	return nullPos
}

func (r *RoutineBook) Acquire(key string) {

	pos := <-r.freeSpaceChan
	r.Set(pos, key)

}

func (r *RoutineBook) FreePage(key string) {

	pos := r.searchPage(key)
	r.Set(pos, emptyKey)
	r.freeSpaceChan <- pos
}

func (r *RoutineBook) Set(pos int, key string) {
	r.Lock()
	r.arr[pos] = key // book page
	r.Unlock()
}

func (r *RoutineBook) ActivePages() int {
	result := 0
	for _, item := range r.arr {
		if item != emptyKey {
			result += 1
		}
	}

	return result
}
