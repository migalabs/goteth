package postgresql

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cortze/eth2-state-analyzer/pkg/metrics"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Static postgres queries, for each modification in the tables, the table needs to be reseted
var (
	// logrus associated with the postgres db
	PsqlType = "postgres-db"
	log      = logrus.WithField(
		"module", PsqlType,
	)
	MAX_BATCH_QUEUE = 300
)

type PostgresDBService struct {
	// Control Variables
	ctx           context.Context
	cancel        context.CancelFunc
	connectionUrl string // the url might not be necessary (better to remove it?¿)
	psqlPool      *pgxpool.Pool

	WriteChan        chan pgx.Batch
	doneTasks        chan interface{}
	endProcess       int32
	FinishSignalChan chan struct{}
	MonitorStruct    *metrics.Monitor
	// Network DB Model
}

// Connect to the PostgreSQL Database and get the multithread-proof connection
// from the given url-composed credentials
func ConnectToDB(ctx context.Context, url string, chanLength int, monitor *metrics.Monitor) (*PostgresDBService, error) {
	mainCtx, cancel := context.WithCancel(ctx)
	// spliting the url to don't share any confidential information on logs
	log.Infof("Conneting to postgres DB %s", url)
	if strings.Contains(url, "@") {
		log.Debugf("Connecting to PostgresDB at %s", strings.Split(url, "@")[1])
	}
	psqlPool, err := pgxpool.Connect(mainCtx, url)
	if err != nil {
		return nil, err
	}
	if strings.Contains(url, "@") {
		log.Infof("PostgresDB %s succesfully connected", strings.Split(url, "@")[1])
	}
	// filter the type of network that we are filtering

	psqlDB := &PostgresDBService{
		ctx:              mainCtx,
		cancel:           cancel,
		connectionUrl:    url,
		psqlPool:         psqlPool,
		WriteChan:        make(chan pgx.Batch, chanLength),
		doneTasks:        make(chan interface{}, 1),
		endProcess:       0,
		FinishSignalChan: make(chan struct{}, 1),
		MonitorStruct:    monitor,
	}
	// init the psql db
	err = psqlDB.init(ctx, psqlDB.psqlPool)
	if err != nil {
		return psqlDB, errors.Wrap(err, "error initializing the tables of the psqldb")
	}
	go psqlDB.runWriters(20)
	return psqlDB, err
}

// Close the connection with the PostgreSQL
func (p *PostgresDBService) Close() {
}

func (p *PostgresDBService) init(ctx context.Context, pool *pgxpool.Pool) error {
	// create the tables
	err := p.createRewardsTable(ctx, pool)
	if err != nil {
		return err
	}

	err = p.createEpochMetricsTable(ctx, pool)
	if err != nil {
		return err
	}

	return nil
}

func (p PostgresDBService) DoneTasks() {
	p.doneTasks <- struct{}{}
}

func (p PostgresDBService) runWriters(workersNum int) {
	var wgDBWriters sync.WaitGroup
	finished := int32(0)
	for i := 0; i < workersNum; i++ {
		wgDBWriters.Add(1)
		go func(dbWriterID int) {
			defer wgDBWriters.Done()
			logWriter := log.WithField("DBWriter", dbWriterID)

			for {

				if p.endProcess >= 1 && len(p.WriteChan) == 0 {
					atomic.AddInt32(&finished, int32(1))
					return
				}
				select {
				case <-p.doneTasks:
					logWriter.Info("finish detected, closing persister")
					atomic.AddInt32(&p.endProcess, int32(1))
				default:
				}

				select {
				case task := <-p.WriteChan:
					logWriter.Debugf("Received new write task")
					err := p.ExecuteBatch(task)
					if err != nil {
						logWriter.Errorf("Error processing batch", err.Error())
					}

				case <-p.ctx.Done():
					logWriter.Info("shutdown detected, closing persister")
					return
				default:
				}

			}

		}(i)
	}

	wgDBWriters.Wait()
	p.FinishSignalChan <- struct{}{}

}

type WriteTask struct {
	QueryID  int
	ModelObj interface{}
}

func (p PostgresDBService) ExecuteBatch(batch pgx.Batch) error {

	snapshot := time.Now()
	tx, err := p.psqlPool.Begin(p.ctx)
	if err != nil {
		panic(err)
	}

	batchResults := tx.SendBatch(p.ctx, &batch)

	var qerr error
	var rows pgx.Rows
	for qerr == nil {
		rows, qerr = batchResults.Query()
		rows.Close()
	}

	p.MonitorStruct.AddDBWrite(time.Since(snapshot).Seconds())
	// log.Debugf("Batch process time: %f", time.Since(snapshot).Seconds())

	return tx.Commit(p.ctx)

}
