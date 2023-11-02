package db

import (
	"context"
	"time"

	pgx "github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	QueryTimeout = 5 * time.Minute
	MaxRetries   = 1

	ErrorNoConnFree = "no connection adquirable"
)

type QueryBatch struct {
	ctx          context.Context
	pgxPool      *pgxpool.Pool
	batch        *pgx.Batch
	size         int
	persistables []Persistable
	metrics      BatchMetrics
}

func NewQueryBatch(ctx context.Context, pgxPool *pgxpool.Pool, batchSize int) *QueryBatch {
	return &QueryBatch{
		ctx:          ctx,
		pgxPool:      pgxPool,
		batch:        &pgx.Batch{},
		size:         batchSize,
		persistables: make([]Persistable, 0),
		metrics:      BatchMetrics{},
	}
}

func (q *QueryBatch) IsReadyToPersist() bool {
	return q.batch.Len() >= q.size
}

func (q *QueryBatch) AddQuery(persis Persistable) {
	q.batch.Queue(persis.query, persis.values...)
	q.persistables = append(q.persistables, persis)
}

func (q *QueryBatch) Len() int {
	return q.batch.Len()
}

func (q *QueryBatch) PersistBatch() error {
	logEntry := log.WithFields(log.Fields{
		"mod": "batch-persister",
	})
	wlog.Tracef("persisting batch of queries with len(%d)", q.Len())
	var err error
persistRetryLoop:
	for i := 0; i < MaxRetries; i++ {
		t := time.Now()
		err = q.persistBatch()
		duration := time.Since(t)
		switch err {
		case nil:
			logEntry.Debugf("persisted %d queries in %s seconds", q.Len(), duration)
			q.metrics.AddPersistStas(duration)
			break persistRetryLoop
		default:
			logEntry.Tracef("attempt numb %d failed %s", i+1, err.Error())
		}
	}
	q.cleanBatch()
	return errors.Wrap(err, "unable to persist batch query")
}

func (q *QueryBatch) persistBatch() error {
	logEntry := log.WithFields(log.Fields{
		"mod": "batch-persister",
	})

	if q.Len() == 0 {
		logEntry.Trace("skipping batch-query, no queries to persist")
		return nil
	}

	ctx, cancel := context.WithTimeout(q.ctx, QueryTimeout)
	defer cancel()

	batchResults := q.pgxPool.SendBatch(ctx, q.batch)
	defer batchResults.Close()

	var qerr error
	var rows pgx.Rows
	nextQuery := true
	cnt := 0
	for nextQuery && qerr == nil {
		rows, qerr = batchResults.Query()
		nextQuery = rows.Next() // it closes all the rows if all the rows are readed
		cnt++
	}
	// check if there was any error
	if qerr != nil {
		log.WithFields(log.Fields{
			"error":  qerr.Error(),
			"query":  q.persistables[cnt-1].query,
			"values": q.persistables[cnt-1].values,
		}).Errorf("unable to persist query [%d]", cnt-1)
		return errors.Wrap(qerr, "error persisting batch")
	}
	if ctx.Err() == context.DeadlineExceeded {
		log.WithFields(log.Fields{
			"error":  ctx.Err().Error(),
			"query":  q.persistables[cnt-1].query,
			"values": q.persistables[cnt-1].values,
		}).Errorf("timed-out [query %d]", cnt-1)
		return errors.Wrap(ctx.Err(), "error persisting batch")
	}
	return nil
}

func (q *QueryBatch) cleanBatch() {
	q.batch = &pgx.Batch{}
	q.persistables = make([]Persistable, 0)
}

// persistable is the main structure fed to the batcher
// allows to link batching errors with the query and values
// that generated it
type Persistable struct {
	query  string
	values []interface{}
}

func NewPersistable() Persistable {
	return Persistable{
		values: make([]interface{}, 0),
	}
}

func (p *Persistable) isEmpty() bool {
	return p.query == ""
}

type BatchMetrics struct {
	numPersists uint64        // number of times this batch persisted
	timeMs      time.Duration // accumulated time this batch has been persisting queries

}

func (b *BatchMetrics) AddPersistStas(newTime time.Duration) {
	b.numPersists += 1
	b.timeMs += newTime
}

func (b BatchMetrics) Average() uint64 { // return miliseconds
	return uint64(b.timeMs.Milliseconds()) / b.numPersists
}
