package dbobs

import (
	"context"
	"fmt"
)

// Target is one inventory row normalized for collectors.
type Target struct {
	Engine   string
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
}

// Collect dispatches to the engine-specific collector. Engines without a
// driver in this build return a descriptive error instead of failing silently.
func Collect(ctx context.Context, target Target, samples *[]QuerySample) (Metrics, error) {
	switch target.Engine {
	case "", "postgres":
		metrics, querySamples, err := CollectPostgres(ctx, DSN(target.Host, target.Port, target.Database, target.Username, target.Password, target.SSLMode))
		*samples = append(*samples, querySamples...)
		return metrics, err
	case "mysql", "mariadb":
		return CollectMySQL(ctx, target, samples)
	case "redis":
		return CollectRedis(ctx, target, samples)
	case "sqlserver":
		return CollectSQLServer(ctx, target, samples)
	case "oracle":
		return CollectOracle(ctx, target, samples)
	case "db2":
		return Metrics{}, fmt.Errorf("DB2 requires the IBM CLI driver; use a TCP monitor for availability for now")
	default:
		return Metrics{}, fmt.Errorf("unknown engine %q", target.Engine)
	}
}
