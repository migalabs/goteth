package metrics

import (
	"sync"
)

type Monitor struct {
	m               sync.Mutex
	DBWriteTime     float64
	DownloadTime    float64
	PreprocessTime  float64
	BatchingTime    float64
	ValidatorLength int
}

func NewMonitorMetrics(valLength int) Monitor {
	return Monitor{
		DBWriteTime:     0,
		DownloadTime:    0,
		PreprocessTime:  0,
		BatchingTime:    0,
		ValidatorLength: valLength,
	}
}

func (p *Monitor) AddDBWrite(executionTime float64) {
	p.m.Lock()
	p.DBWriteTime += executionTime

	p.m.Unlock()
}

func (p *Monitor) AddDownload(executionTime float64) {
	p.m.Lock()
	p.DownloadTime += executionTime

	p.m.Unlock()
}

func (p *Monitor) AddPreprocessTime(executionTime float64) {
	p.m.Lock()
	p.PreprocessTime += executionTime

	p.m.Unlock()
}

func (p *Monitor) AddBatchingTime(executionTime float64) {
	p.m.Lock()
	p.BatchingTime += executionTime

	p.m.Unlock()
}
