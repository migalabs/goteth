package db

import (
	"context"
	"fmt"

	"sync"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/sirupsen/logrus"
)

var (
	modName  = "db"
	PsqlType = "clickhouse-db"
	log      = logrus.WithField(
		"module", modName,
	)
	MAX_BATCH_QUEUE       = 1000
	MAX_EPOCH_BATCH_QUEUE = 1
)

type DBServiceOption func(*DBService) error

type DBService struct {
	// Control Variables
	ctx           context.Context
	connectionUrl string // the url might not be necessary (better to remove it?¿)

	lowLevelClient  *ch.Client  // for bulk loads, mainly insert
	highLevelClient driver.Conn // for side tasks, like Select and Delete

	monitorMetrics map[string][]DBMonitorMetrics // map table and metrics
	lowMu          sync.Mutex
	highMu         sync.Mutex
}

func New(ctx context.Context, url string, options ...DBServiceOption) (*DBService, error) {
	var err error
	pService := &DBService{
		ctx:            ctx,
		connectionUrl:  url,
		monitorMetrics: make(map[string][]DBMonitorMetrics),
	}

	for _, o := range options {
		err := o(pService)
		if err != nil {
			return pService, err
		}
	}

	return pService, err
}

func (s *DBService) Connect() error {
	err := s.ConnectLowLevel()
	if err != nil {
		return err
	}

	err = s.ConnectHighLevel()
	if err != nil {
		return err
	}
	return nil

}

func WithUrl(url string) DBServiceOption {
	return func(s *DBService) error {
		s.connectionUrl = url
		return nil
	}
}

func (p *DBService) Finish() {

	p.lowLevelClient.Close()
	p.highLevelClient.Close()
	log.Infof("Routines finished...")
	log.Infof("closing connection to database server...")
	log.Infof("connection to database server closed...")
}

type DBMonitorMetrics struct {
	Rows        int           // how many rows were persisted in the last copy
	PersistTime time.Duration // how much time to persist the last copy
}

type DeletableObject struct {
	query string
	table string
	args  []any
}

func NewDeletableObj(query string, table string, args []any) DeletableObject {
	return DeletableObject{
		query: query,
		table: table,
		args:  args,
	}
}

func (d DeletableObject) Query() string {
	return fmt.Sprintf(d.query, d.table)
}
func (d DeletableObject) Table() string {
	return d.table
}
func (d DeletableObject) Args() []any {
	return d.args
}

type Input[T any] func(t T) proto.Input

type PersistableObject[
	T spec.AgnosticBlock |
		spec.Epoch |
		api.FinalizedCheckpointEvent |
		int64 |
		spec.ProposerDuty |
		api.ChainReorgEvent |
		spec.AgnosticTransaction |
		spec.ValidatorLastStatus |
		spec.ValidatorRewards |
		spec.Withdrawal |
		HeadEvent] struct {
	table string
	query string
	data  []T
	input Input[[]T]
}

func (d *PersistableObject[T]) Append(newData T) {
	d.data = append(d.data, newData)
}

func (d PersistableObject[T]) Table() string {
	return d.table
}

func (d PersistableObject[T]) Columns() int {
	return len(d.Input().Columns())
}

func (d PersistableObject[T]) Rows() int {
	return len(d.data)
}

func (d PersistableObject[T]) Query() string {
	return fmt.Sprintf(d.query, d.table)
}

func (d PersistableObject[T]) Input() proto.Input {
	return d.input(d.data)
}

func (d PersistableObject[T]) ExportPersist() (string, string, proto.Input, int) {
	return d.Query(), d.Table(), d.Input(), d.Rows()
}
