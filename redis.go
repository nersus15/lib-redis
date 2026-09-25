package redis

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/webcore-go/webcore/infra/config"
)

// Redis represents shared Redis connection
type Redis struct {
	Client *redis.Client
}

// NewRedis creates a new Redis connection
func NewRedis(config config.RedisConfig) *Redis {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	return &Redis{Client: client}
}

func (r *Redis) Install(args ...any) error {
	// Tidak melakukan apa-apa
	return nil
}

func (r *Redis) Connect() error {
	// Test connection
	_, err := r.Client.Ping(r.Client.Context()).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return nil
}

// Close closes the Redis connection
func (r *Redis) Disconnect() error {
	return r.Client.Close()
}

func (r *Redis) Uninstall() error {
	// Tidak melakukan apa-apa
	return nil
}

func (r *Redis) Set(key string, value any, ttl time.Duration) error {
	ctx := r.Client.Context()

	var val string

	// 1. Dereference jika value berupa pointer (seperti pada MemoryCache)
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return fmt.Errorf("cannot set nil pointer value for key %s", key)
		}
		value = rv.Elem().Interface()
	}

	// 2. Format tipe data primitif ke string, sisanya ke JSON
	switch v := value.(type) {
	case string:
		val = v
	case int:
		val = strconv.FormatInt(int64(v), 10)
	case int8:
		val = strconv.FormatInt(int64(v), 10)
	case int16:
		val = strconv.FormatInt(int64(v), 10)
	case int32:
		val = strconv.FormatInt(int64(v), 10)
	case int64:
		val = strconv.FormatInt(v, 10)
	case uint:
		val = strconv.FormatUint(uint64(v), 10)
	case uint8:
		val = strconv.FormatUint(uint64(v), 10)
	case uint16:
		val = strconv.FormatUint(uint64(v), 10)
	case uint32:
		val = strconv.FormatUint(uint64(v), 10)
	case uint64:
		val = strconv.FormatUint(v, 10)
	case bool:
		val = "0"
		if v {
			val = "1"
		}
	case float32:
		val = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		val = strconv.FormatFloat(v, 'f', -1, 64)
	default:
		bytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache value for key %s: %w", key, err)
		}
		val = string(bytes)
	}

	return r.Client.Set(ctx, key, val, ttl).Err()
}

func (r *Redis) Get(key string, outvalue any) bool {
	ctx := r.Client.Context()

	val, err := r.Client.Get(ctx, key).Result()
	if err == redis.Nil || err != nil {
		return false
	}

	// Validasi outvalue harus berupa pointer dan tidak nil
	rv := reflect.ValueOf(outvalue)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return false
	}

	elem := rv.Elem()

	switch elem.Kind() {
	case reflect.String:
		elem.SetString(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			elem.SetInt(i)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			elem.SetUint(i)
		}
	case reflect.Bool:
		elem.SetBool(val == "1")
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			elem.SetFloat(f)
		}
	default:
		// Tipe kompleks (struct/map/slice) di-unmarshal dari JSON string
		err := json.Unmarshal([]byte(val), outvalue)
		if err != nil {
			return false
		}
	}

	return true
}
func (r *Redis) Delete(key string) error {
	ctx := r.Client.Context()

	return r.Client.Del(ctx, key).Err()
}
