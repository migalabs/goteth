package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Static postgres queries, for each modification in the tables, the table needs to be reseted
var (
	// wlogrus associated with the postgres db
	PsqlType = "postgres-db"
	wlog     = logrus.WithField(
		"module", PsqlType,
	)
	MAX_BATCH_QUEUE       = 1000
	MAX_EPOCH_BATCH_QUEUE = 1
)

type PostgresDBService struct {
	// Control Variables
	ctx           context.Context
	connectionUrl string // the url might not be necessary (better to remove it?¿)
	psqlPool      *pgxpool.Pool
	wgDBWriters   sync.WaitGroup

	dummy bool // For testing purposes, when we want to simulate a database service

	writeChan chan Model // Receive tasks to persist
	stop      bool
	workerNum int
}

func New(options ...func(*PostgresDBService)) (*PostgresDBService, error) {
	var err error
	pService := &PostgresDBService{}
	for _, o := range options {
		o(pService)
	}
	if pService.workerNum < 1 {
		return nil, fmt.Errorf("worker num must be over 0")
	}

	pService.writeChan = make(chan Model, pService.workerNum)

	if pService.dummy {
		pService.psqlPool = &pgxpool.Pool{}
		go pService.runDummywriter()
		return pService, nil
	}

	err = pService.ConnectToDB()

	return pService, err
}

func WithContext(ctx context.Context) func(*PostgresDBService) {
	return func(s *PostgresDBService) {
		s.ctx = ctx
	}
}

func WithUrl(url string) func(*PostgresDBService) {
	return func(s *PostgresDBService) {
		s.connectionUrl = url
	}
}

func WithWorkers(workerNum int) func(*PostgresDBService) {
	return func(s *PostgresDBService) {
		s.workerNum = workerNum
	}
}

func WithDummyPersister(dummy bool) func(*PostgresDBService) {
	return func(s *PostgresDBService) {
		s.dummy = dummy
	}
}

// Connect to the PostgreSQL Database and get the multithread-proof connection
// from the given url-composed credentials
func (s *PostgresDBService) ConnectToDB() error {
	// spliting the url to don't share any confidential information on wlogs
	wlog.Infof("Connecting to postgres DB %s", s.connectionUrl)
	psqlPool, err := pgxpool.Connect(s.ctx, s.connectionUrl)
	if err != nil {
		return errors.Wrap(err, "error connecting to the psqldb")
	}
	s.psqlPool = psqlPool
	wlog.Infof("successfully connected to the database")

	// init the psql db
	err = s.init(s.ctx, s.psqlPool)
	if err != nil {
		return errors.Wrap(err, "error initializing the tables of the psqldb")
	}
	go s.runWriters()
	return err
}

func (p *PostgresDBService) init(ctx context.Context, pool *pgxpool.Pool) error {
	// create the tables
	err := p.createRewardsTable()
	if err != nil {
		return err
	}

	err = p.createEpochMetricsTable()
	if err != nil {
		return err
	}

	err = p.createProposerDutiesTable()
	if err != nil {
		return err
	}

	err = p.createBlockMetricsTable()
	if err != nil {
		return err
	}

	err = p.createStatusTable()
	if err != nil {
		return err
	}

	err = p.createPoolsTable()
	if err != nil {
		return err
	}

	err = p.createWithdrawalsTable()
	if err != nil {
		return err
	}

	err = p.createLastStatusTable()
	if err != nil {
		return err
	}

	err = p.createTransactionsTable()
	if err != nil {
		return err
	}

	return nil
}

func (p *PostgresDBService) Finish() {
	p.stop = true
	p.wgDBWriters.Wait()
	wlog.Infof("Routines finished...")
	wlog.Infof("closing connection to database server...")
	if !p.dummy {
		p.psqlPool.Close()
	}
	wlog.Infof("connection to database server closed...")
	close(p.writeChan)
}

func (p *PostgresDBService) runWriters() {

	wlog.Info("Launching Beacon State Writers")
	wlog.Infof("Launching %d Beacon State Writers", p.workerNum)
	for i := 0; i < p.workerNum; i++ {
		p.wgDBWriters.Add(1)
		go func(dbWriterID int) {
			defer p.wgDBWriters.Done()
			batcher := NewQueryBatch(p.ctx, p.psqlPool, MAX_BATCH_QUEUE)
			wlogWriter := wlog.WithField("DBWriter", dbWriterID)
			ticker := time.NewTicker(utils.RoutineFlushTimeout)
		loop:
			for {
				select {
				case task := <-p.writeChan:

					var err error
					persis := NewPersistable()

					switch task.Type() {
					case spec.BlockModel:
						q, args := BlockOperation(task.(spec.AgnosticBlock))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.EpochModel:
						q, args := EpochOperation(task.(spec.Epoch))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.PoolSummaryModel:
						q, args := PoolOperation(task.(spec.PoolSummary))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.ProposerDutyModel:
						q, args := ProposerDutyOperation(task.(spec.ProposerDuty))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.ValidatorLastStatusModel:
						q, args := ValidatorLastStatusOperation(task.(spec.ValidatorLastStatus))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.ValidatorRewardsModel:
						q, args := ValidatorOperation(task.(spec.ValidatorRewards))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.WithdrawalModel:
						q, args := WithdrawalOperation(task.(spec.Withdrawal))
						persis.query = q
						persis.values = append(persis.values, args...)
					case spec.TransactionsModel:
						q, args := TransactionOperation(task.(*spec.AgnosticTransaction))
						persis.query = q
						persis.values = append(persis.values, args...)
					default:
						err = fmt.Errorf("could not figure out the type of write task")
						wlog.Errorf("could not process incoming task, %s", err)
					}
					// ckeck if there is any new query to add
					if !persis.isEmpty() {
						batcher.AddQuery(persis)
					}
					// check if we can flush the batch of queries
					if batcher.IsReadyToPersist() {
						err := batcher.PersistBatch()
						if err != nil {
							wlogWriter.Errorf("Error processing batch", err.Error())
						}
					}

				case <-p.ctx.Done():
					break loop

				case <-ticker.C:
					// if limit reached or no more queue and pending tasks
					if batcher.IsReadyToPersist() || (len(p.writeChan) == 0 && batcher.Len() > 0) {
						wlog.Tracef("flushing batcher")
						err := batcher.PersistBatch()
						if err != nil {
						}
					}

					if p.stop && len(p.writeChan) == 0 {
						break loop
					}
				}
			}
		}(i)
	}

}

func (p *PostgresDBService) runDummywriter() {

	ticker := time.NewTicker(utils.RoutineFlushTimeout)
	for {
		select {
		case <-p.writeChan:
			continue
		case <-p.ctx.Done():
			return

		case <-ticker.C:

			if p.stop && len(p.writeChan) == 0 {
				return
			}
		}
	}

}

func (p *PostgresDBService) Persist(w Model) {
	p.writeChan <- w
}

type Model interface { // simply to enforce a Model interface
	// For now we simply support insert operations
	Type() spec.ModelType // whether insert is activated for this model
}
