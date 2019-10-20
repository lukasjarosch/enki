package mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/database/mysql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/jmoiron/sqlx"
)

type MySQL struct {
	db   *sqlx.DB
	dsn  string
	opts *Options
}

const (
	DefaultMigrationPath         = "migrations"
	DefaultMaxOpenConnections    = 10
	DefaultMaxIdleConnections    = 0
	DefaultMaxConnectionLifetime = 600 * time.Second
	DriverName                   = "mysql"
)

// New will connect to the MySQL server using the given DSN
func New(dsn string, options ...Option) (*MySQL, error) {
	args := &Options{
		MigrationPath:         DefaultMigrationPath,
		MaxOpenConnections:    DefaultMaxOpenConnections,
		MaxIdleConnections:    DefaultMaxIdleConnections,
		MaxConnectionLifetime: DefaultMaxConnectionLifetime,
	}

	for _, opt := range options {
		opt(args)
	}

	db, err := sqlx.Connect(DriverName, dsn)
	if err != nil {
		return nil, err
	}

	// configure connection pool
	db.SetMaxOpenConns(args.MaxOpenConnections)
	db.SetMaxIdleConns(args.MaxIdleConnections)
	db.SetConnMaxLifetime(args.MaxConnectionLifetime)

	return &MySQL{
		db:   db,
		dsn:  dsn,
		opts: args,
	}, nil
}

// Migrate to a specific version. The migrations need to be placed in the MigrationPath.
// For every change, two migrations should be created:
// 		1_add_example_table.up.sql
// 		1_add_example_table.down.sql
func (m MySQL) Migrate(version uint) error {
	db, err := sql.Open(DriverName, m.dsn)
	if err != nil {
		return err
	}
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	migrations, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", m.opts.MigrationPath),
		DriverName,
		driver)
	if err != nil {
		return err
	}

	err = migrations.Migrate(version)
	if err != nil {
		if strings.Contains(err.Error(), "no change") {
			return nil
		}
		return err
	}

	return nil
}

// Close is just a proxy for convenient access to db.Close()
func (m MySQL) Close() error {
	return m.db.Close()
}

// DB is just a proxy for convenient access to the underlying sqlx implementation
// This method is used a lot, therefore it's name is abbreviated.
func (m MySQL) DB() *sqlx.DB {
	return m.db
}