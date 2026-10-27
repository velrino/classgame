package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

type Service struct {
	client *goredis.Client
}

func NewService(client *goredis.Client) *Service {
	return &Service{client: client}
}

func (service *Service) Ping(ctx context.Context) error {
	return service.client.Ping(ctx).Err()
}
