package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var ErrMissingURL = errors.New("redis url is required")

const DefaultPingTimeout = 5 * time.Second

type Options struct {
	URL         string
	CACertPath  string
	CACertPEM   string
	PingTimeout time.Duration
}

type Connection struct {
	client  *goredis.Client
	service *Service
	enabled bool
}

func Connect(ctx context.Context, input Options) (*Connection, error) {
	if input.URL == "" {
		return nil, ErrMissingURL
	}

	resolved := resolveOptions(input)

	clientOptions, err := goredis.ParseURL(resolved.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	tlsConfig, err := buildTLSConfig(resolved)
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		clientOptions.TLSConfig = mergeTLSConfig(clientOptions.TLSConfig, tlsConfig)
	}

	client := goredis.NewClient(clientOptions)
	service := NewService(client)

	pingContext, cancel := context.WithTimeout(ctx, resolved.PingTimeout)
	defer cancel()

	if err := service.Ping(pingContext); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Connection{
		client:  client,
		service: service,
		enabled: true,
	}, nil
}

func (connection *Connection) IsEnabled() bool {
	return connection != nil && connection.enabled
}

func (connection *Connection) Client() *goredis.Client {
	if connection == nil {
		return nil
	}

	return connection.client
}

func (connection *Connection) Service() *Service {
	if connection == nil {
		return nil
	}

	return connection.service
}

func (connection *Connection) Ping(ctx context.Context) error {
	if !connection.IsEnabled() {
		return ErrMissingURL
	}

	return connection.service.Ping(ctx)
}

func (connection *Connection) Disconnect(_ context.Context) error {
	if !connection.IsEnabled() {
		return nil
	}

	connection.enabled = false
	return connection.client.Close()
}

func resolveOptions(input Options) Options {
	if input.PingTimeout <= 0 {
		input.PingTimeout = DefaultPingTimeout
	}

	return input
}
