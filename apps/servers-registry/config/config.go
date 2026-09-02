package config

import (
	"strconv"

	"github.com/walkline/ToCloud9/shared/config"
)

// Config is config of application
type Config struct {
	config.Logging `yaml:"logging"`

	// Port is port that would be used for grpc server
	Port string `yaml:"port" env:"PORT" env-default:"8999"`

	// HealthCheckPort is the port that serves /healthcheck for k8s
	// liveness/readiness probes of the registry process itself. It reports
	// unhealthy once HealthCheckGracePeriodSecs has elapsed since startup
	// and no game server is registered.
	HealthCheckPort string `yaml:"healthCheckPort" env:"HEALTH_CHECK_PORT" env-default:"8900"`

	// HealthCheckGracePeriodSecs is how long after startup the registry
	// keeps reporting healthy even with zero registered game servers. This
	// covers the normal cold-start window where world servers are still
	// booting (loading maps, DB pools, etc.) and haven't registered yet, so
	// k8s doesn't restart the registry before they've had a fair chance.
	HealthCheckGracePeriodSecs int `yaml:"healthCheckGracePeriodSecs" env:"HEALTH_CHECK_GRACE_PERIOD_SECS" env-default:"180"`

	// RedisConnection is connection string for the redis connection
	RedisConnection string `yaml:"redisUrl" env:"REDIS_URL" env-default:"redis://:@redis:6379/0"`

	// NatsURL is nats connection url
	NatsURL string `yaml:"natsUrl" env:"NATS_URL" env-default:"nats://nats:4222"`

	// HealthCheckMaxFails is the number of consecutive failed health checks a game/gateway server must accumulate before it's evicted from the registry.
	HealthCheckMaxFails int `yaml:"healthCheckMaxFails" env:"HEALTH_CHECK_MAX_FAILS" env-default:"3"`

	// RealmsIDs is id of realms that the system supports.
	RealmsID []uint32 `yaml:"realmsID" env:"REALMs_ID" env-default:"1"`

	Layering LayeringConfig `yaml:"layering"`
}

func (c Config) HealthCheckPortInt() (p int) {
	p, _ = strconv.Atoi(c.HealthCheckPort)
	return
}

type LayeringConfig struct {
	Maps map[uint32]uint32 `yaml:"maps" env:"LAYER_MAPS" env-separator:";"`
}

// LoadConfig loads config from env variables
func LoadConfig() (*Config, error) {
	var c struct {
		Root Config `yaml:"servers-registry"`
	}

	err := config.LoadConfig(&c)
	if err != nil {
		return nil, err
	}

	return &c.Root, nil
}
