package common

import (
	"crypto/tls"
	"time"

	"github.com/RichardKnop/machinery/v1/config"
	"github.com/gomodule/redigo/redis"
)

var (
	defaultConfig = &config.RedisConfig{
		MaxIdle:                3,
		IdleTimeout:            240,
		ReadTimeout:            15,
		WriteTimeout:           15,
		ConnectTimeout:         15,
		DelayedTasksPollPeriod: 20,
		NormalTasksPollPeriod:  15, // Affected by ReadTimeout
	}
)

// RedisConnector ...
type RedisConnector struct{}

// NewPool returns a new pool of Redis connections
func (rc *RedisConnector) NewPool(socketPath, host, password string, db int, cnf *config.RedisConfig, tlsConfig *tls.Config) *redis.Pool {
	if cnf == nil {
		cnf = defaultConfig
	}
	return &redis.Pool{
		MaxIdle:     cnf.MaxIdle,
		IdleTimeout: time.Duration(cnf.IdleTimeout) * time.Second,
		MaxActive:   cnf.MaxActive,
		Wait:        cnf.Wait,
		Dial: func() (redis.Conn, error) {
			c, err := rc.open(socketPath, host, password, db, cnf, tlsConfig)
			if err != nil {
				return nil, err
			}

			if db != 0 {
				_, err = c.Do("SELECT", db)
				if err != nil {
					return nil, err
				}
			}

			return c, err
		},
		// PINGs connections that have been idle more than 10 seconds
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Duration(10*time.Second) {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

// Open a new Redis connection
func (rc *RedisConnector) open(socketPath, host, password string, db int, cnf *config.RedisConfig, tlsConfig *tls.Config) (redis.Conn, error) {
	var opts = []redis.DialOption{
		redis.DialDatabase(db),
		redis.DialReadTimeout(time.Duration(cnf.ReadTimeout) * time.Second),
		redis.DialWriteTimeout(time.Duration(cnf.WriteTimeout) * time.Second),
		redis.DialConnectTimeout(time.Duration(cnf.ConnectTimeout) * time.Second),
	}

	if tlsConfig != nil {
		opts = append(opts, redis.DialTLSConfig(tlsConfig), redis.DialUseTLS(true))
	}

	if password != "" {
		opts = append(opts, redis.DialPassword(password))
	}

	if socketPath != "" {
		return redis.Dial("unix", socketPath, opts...)
	}

	return redis.Dial("tcp", host, opts...)
}
