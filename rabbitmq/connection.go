package rabbitmq

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

// Connection is a wrapper for amqp.Connection but adding reconnection functionality.
type Connection struct {
	addr                  string
	conn                  *amqp.Connection
	connMutex             sync.Mutex
	logger                *zap.Logger
	ctx                   context.Context
	cancel                context.CancelFunc
	connected             bool
	notifyCloseConnection chan *amqp.Error
}

const ReconnectDelay = 5 * time.Second

func NewConnection(addr string, logger *zap.Logger) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Connection{
		ctx:                   ctx,
		logger:                logger,
		cancel:                cancel,
		addr:                  addr,
		connMutex:             sync.Mutex{},
		notifyCloseConnection: make(chan *amqp.Error),
	}

	return c
}

// Connect will dial to the specified AMQP server addr.
func (c *Connection) Connect() (err error) {
	c.conn, err = c.dial()
	if err != nil {
		return errors.Wrap(err, "unable to connect to amqp server")
	}

	go c.monitorConnection()

	return nil
}

// Shutdown the reconnector and terminate any existing connections
func (c *Connection) Shutdown() {
	c.setConnected(false)
	c.cancel()

	if c.IsConnected() {
		err := c.conn.Close()
		if err != nil {
			c.logger.Warn("error while closing amqp connection", zap.Error(err))
			return
		}
	}
}

// dial and return the connection and any occurred error
func (c *Connection) dial() (*amqp.Connection, error) {
	c.setConnected(false)

	conn, err := amqp.Dial(c.addr)
	if err != nil {
		return nil, err
	}
	c.changeConnection(conn)
	c.setConnected(true)
	return conn, nil
}

// monitorConnection ensures that the amqp connection is recovered on failures.
// if an error can be read from the amqp connectionClosed channel, then reconnect() is called
func (c *Connection) monitorConnection() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case amqpErr, ok := <-c.notifyCloseConnection:
			if ok {
				c.logger.Warn("amqp connection error",
					zap.String("err.reason", amqpErr.Reason),
					zap.Int("err.code", amqpErr.Code))
				c.reconnect()
			}
		}
	}
}

// reconnect will, once started, try to connect to amqp forever
// the method only returns if a connection is established or the ctxReconnect context is cancelled by Shutdown()
func (c *Connection) reconnect() {
	var err error

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		c.conn, err = c.dial()
		if err != nil {
			c.logger.Warn("unable to connect to amqp server", zap.Error(err))
			time.Sleep(ReconnectDelay)
			continue
		}
		c.logger.Info("reconnected to amqp server")
		c.setConnected(true)
		return
	}
}

// changeConnection sets a new amqp.Connection and renews the notification channel
func (c *Connection) changeConnection(connection *amqp.Connection) {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	c.conn = connection
	c.notifyCloseConnection = make(chan *amqp.Error)
	c.conn.NotifyClose(c.notifyCloseConnection)
}

func (c *Connection) IsConnected() bool {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	return c.connected
}

func (c *Connection) setConnected(status bool) {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	c.connected = status
}

func (c *Connection) Channel() (*amqp.Channel, error) {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	return c.conn.Channel()
}
