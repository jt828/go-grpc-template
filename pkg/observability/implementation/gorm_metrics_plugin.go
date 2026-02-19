package implementation

import (
	"context"
	"time"

	"github.com/jt828/go-grpc-template/pkg/observability"
	"gorm.io/gorm"
)

const metricsStartTimeKey = "metrics:start_time"

type GormMetricsPlugin struct {
	queryLatency observability.Histogram
	queryTotal   observability.Counter
	queryErrors  observability.Counter
}

func NewGormMetricsPlugin(meter observability.Meter) *GormMetricsPlugin {
	return &GormMetricsPlugin{
		queryLatency: meter.Histogram("gorm_query_duration_seconds", observability.MetricOpt{
			Help:      "Duration of GORM queries in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			LabelKeys: []string{"operation"},
		}),
		queryTotal: meter.Counter("gorm_query_total", observability.MetricOpt{
			Help:      "Total number of GORM queries",
			LabelKeys: []string{"operation"},
		}),
		queryErrors: meter.Counter("gorm_query_errors_total", observability.MetricOpt{
			Help:      "Total number of GORM query errors",
			LabelKeys: []string{"operation"},
		}),
	}
}

func (p *GormMetricsPlugin) Name() string {
	return "metrics"
}

func (p *GormMetricsPlugin) Initialize(db *gorm.DB) error {
	db.Callback().Create().Before("gorm:create").Register("metrics:before_create", p.before)
	db.Callback().Create().After("gorm:create").Register("metrics:after_create", p.after("create"))

	db.Callback().Query().Before("gorm:query").Register("metrics:before_query", p.before)
	db.Callback().Query().After("gorm:query").Register("metrics:after_query", p.after("query"))

	db.Callback().Update().Before("gorm:update").Register("metrics:before_update", p.before)
	db.Callback().Update().After("gorm:update").Register("metrics:after_update", p.after("update"))

	db.Callback().Delete().Before("gorm:delete").Register("metrics:before_delete", p.before)
	db.Callback().Delete().After("gorm:delete").Register("metrics:after_delete", p.after("delete"))

	db.Callback().Row().Before("gorm:row").Register("metrics:before_row", p.before)
	db.Callback().Row().After("gorm:row").Register("metrics:after_row", p.after("row"))

	db.Callback().Raw().Before("gorm:raw").Register("metrics:before_raw", p.before)
	db.Callback().Raw().After("gorm:raw").Register("metrics:after_raw", p.after("raw"))

	return nil
}

func (p *GormMetricsPlugin) before(db *gorm.DB) {
	db.Statement.Context = context.WithValue(db.Statement.Context, metricsStartTimeKey, time.Now())
}

func (p *GormMetricsPlugin) after(operation string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		opLabel := observability.Label{Key: "operation", Value: operation}

		p.queryTotal.Inc(1, opLabel)

		if db.Error != nil {
			p.queryErrors.Inc(1, opLabel)
		}

		startTime, ok := db.Statement.Context.Value(metricsStartTimeKey).(time.Time)
		if ok {
			duration := time.Since(startTime).Seconds()
			p.queryLatency.Observe(duration, opLabel)
		}
	}
}
