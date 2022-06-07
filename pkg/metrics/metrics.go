package metrics

import (
	"context"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/metric/unit"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.uber.org/zap"
)

var (
	FrameAge          syncint64.Histogram
	InferenceDuration syncint64.Histogram
)

func initMeter(ctx context.Context) func() {
	zap.S().Info("init telemetry")
	exporter, err := stdout.New(
		stdout.WithPrettyPrint(),
	)
	if err != nil {
		zap.S().Panicf("failed to initialize prometheus exporter %v", err)
	}

	pusher := controller.New(
		processor.NewFactory(
			simple.NewWithInexpensiveDistribution(),
			//simple.NewWithHistogramDistribution(
			//	histogram.WithExplicitBoundaries(
			//		[]float64{.005, .5, 1, 2.5, 5, 10, 20, 50, 100},
			//	),
			//),
			exporter,
		),
		controller.WithExporter(exporter),
	)

	if err = pusher.Start(ctx); err != nil {
		zap.S().Fatalf("starting push controller: %v", err)
	}
	global.SetMeterProvider(pusher)
	return func() {
		if err := pusher.Stop(ctx); err != nil {
			zap.S().Fatalf("stopping push controller: %v", err)
		}
	}

}

func Init(ctx context.Context) func() {
	cleaner := initMeter(ctx)
	var err error

	meter := global.Meter("robocar/rc-steering")
	FrameAge, err = meter.SyncInt64().Histogram(
		"robocar.frame_age",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("time before frame processing"),
	)
	if err != nil {
		zap.S().Panicf("unable to instantiate FrameAge histogram: %v", err)
	}
	InferenceDuration, err = meter.SyncInt64().Histogram(
		"robocar.inference_duration",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("tensorflow inference duration"),
	)
	if err != nil {
		zap.S().Panicf("unable to instantiate InferenceDuration histogram: %v", err)
	}

	return cleaner
}
