package mysql

import (
	"time"
)

type Options struct {
	MigrationPath         string
	MaxOpenConnections    int
	MaxIdleConnections    int
	MaxConnectionLifetime time.Duration
}

type Option func(*Options)

func MigrationPath(path string) Option {
	return func(options *Options) {
		options.MigrationPath = path
	}
}

func MaxOpenConnections(connLimit int) Option {
	return func(options *Options) {
		options.MaxOpenConnections = connLimit
	}
}

func MaxIdleConnections(connLimit int) Option {
	return func(options *Options) {
		options.MaxIdleConnections = connLimit
	}
}

func MaxConnectionLifetime(maxLifetime time.Duration) Option {
	return func(options *Options) {
		options.MaxConnectionLifetime = maxLifetime
	}
}