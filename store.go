package ecaptcha

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix     = "ecaptcha:"
	attemptsKey   = "ecaptcha:attempts:"
)

// RedisStore Redis 存储实现
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建 Redis 存储
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Set 存储验证数据
func (s *RedisStore) Set(ctx context.Context, id string, data []byte, expire time.Duration) error {
	return s.client.Set(ctx, keyPrefix+id, data, expire).Err()
}

// Get 获取验证数据
func (s *RedisStore) Get(ctx context.Context, id string) ([]byte, error) {
	data, err := s.client.Get(ctx, keyPrefix+id).Bytes()
	if err == redis.Nil {
		return nil, ErrCaptchaNotFound
	}
	return data, err
}

// Delete 删除验证数据
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	return s.client.Del(ctx, keyPrefix+id).Err()
}

// IncrAttempts 增加尝试次数
func (s *RedisStore) IncrAttempts(ctx context.Context, id string) (int, error) {
	key := attemptsKey + id
	count, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// 设置过期时间 (与验证码同步过期)
	if count == 1 {
		s.client.Expire(ctx, key, 5*time.Minute)
	}
	return int(count), nil
}

// MemoryStore 内存存储 (用于测试)
type MemoryStore struct {
	data     map[string][]byte
	attempts map[string]int
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:     make(map[string][]byte),
		attempts: make(map[string]int),
	}
}

// Set 存储验证数据
func (s *MemoryStore) Set(ctx context.Context, id string, data []byte, expire time.Duration) error {
	s.data[keyPrefix+id] = data
	return nil
}

// Get 获取验证数据
func (s *MemoryStore) Get(ctx context.Context, id string) ([]byte, error) {
	data, ok := s.data[keyPrefix+id]
	if !ok {
		return nil, ErrCaptchaNotFound
	}
	return data, nil
}

// Delete 删除验证数据
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	delete(s.data, keyPrefix+id)
	return nil
}

// IncrAttempts 增加尝试次数
func (s *MemoryStore) IncrAttempts(ctx context.Context, id string) (int, error) {
	s.attempts[id]++
	return s.attempts[id], nil
}
