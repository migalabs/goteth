package db

import (
	"context"
	
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/migalabs/goteth/pkg/utils"
)

func (s *DBService) ConnectHighLevel() error {
	opts := ParseChUrlIntoOptionsHighLevel(s.connectionUrl)
	conn, err := clickhouse.Open(&opts)
	if err != nil {
		return err
	}
	s.highLevelClient = conn
	return conn.Ping(context.Background())

}

func ParseChUrlIntoOptionsHighLevel(url string) clickhouse.Options {
	var user string
	var password string
	var database string

	protocolAndDetails := strings.Split(url, "://")
	// protocol := protocolAndDetails[0]
	details := protocolAndDetails[1]

	credentialsAndEndpoint := strings.Split(details, "@")
	credentials := credentialsAndEndpoint[0]
	endpoint := credentialsAndEndpoint[1]

	hostPortAndPathParams := strings.Split(endpoint, "/")
	fqdn := hostPortAndPathParams[0]
	pathParams := hostPortAndPathParams[1]

	pathAndParams := strings.Split(pathParams, "?")
	database = pathAndParams[0]
	// params := pathAndParams[1]

	user = strings.Split(credentials, ":")[0]
	password = strings.Split(credentials, ":")[1]

	return clickhouse.Options{
		Addr: []string{fqdn},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		TLS: &tls.Config{
			InsecureSkipVerify: false,
		},
		Debug: false,
		Debugf: func(format string, v ...any) {
			fmt.Printf(format, v)
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:          time.Second * 30,
		MaxOpenConns:         5,
		MaxIdleConns:         5,
		ConnMaxLifetime:      time.Duration(10) * time.Minute,
		ConnOpenStrategy:     clickhouse.ConnOpenInOrder,
		BlockBufferSize:      10,
		MaxCompressionBuffer: 10240,
		ClientInfo: clickhouse.ClientInfo{ // optional, please see Client info section in the README.md
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: utils.CliName, Version: utils.Version},
			},
		}}
}

func (p *DBService) Delete(obj DeletableObject) error {

	var err error
	startTime := time.Now()

	p.highMu.Lock()
	err = p.highLevelClient.Exec(p.ctx, obj.Query(), obj.Args()...)
	p.highMu.Unlock()

	if err == nil {
		log.Infof("query: %s finished in %f seconds", obj.Query(), time.Since(startTime).Seconds())
	}

	return err
}

func (p *DBService) highSelect(query string, dest interface{}) error {
	startTime := time.Now()
	p.highMu.Lock()
	err := p.highLevelClient.Select(p.ctx, dest, query)
	p.highMu.Unlock()

	if err == nil {
		log.Debugf("retrieved %d rows in %f seconds, query: %s", 1, time.Since(startTime).Seconds(), query)
	} else {
		log.Errorf("error executing %s: %s", query, err)
	}

	return err
}
