package logger

import (
	"os"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger adapts zap.Logger to our Logger interface
type ZapLogger struct {
	logger *zap.Logger
	config *types.LoggerConfig
}

// NewZapLogger creates a new ZapLogger with optional configuration
func NewZapLogger(logger *zap.Logger, config *types.LoggerConfig) types.Logger {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	if config == nil {
		config = &types.LoggerConfig{
			MinLevel:       types.LogLevelInfo,
			EnableSampling: false,
		}
	}
	return &ZapLogger{
		logger: logger,
		config: config,
	}
}

// convertToZapFields converts our LogFields to zap.Fields
func convertToZapFields(fields []types.LogField) []zap.Field {
	if len(fields) == 0 {
		return nil
	}
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = zap.Any(f.Key, f.Value)
	}
	return zapFields
}

// convertLogLevel converts our LogLevel to zapcore.Level
func convertLogLevel(level types.LogLevel) zapcore.Level {
	switch level {
	case types.LogLevelDebug:
		return zapcore.DebugLevel
	case types.LogLevelInfo:
		return zapcore.InfoLevel
	case types.LogLevelWarn:
		return zapcore.WarnLevel
	case types.LogLevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func (l *ZapLogger) shouldLog(level types.LogLevel) bool {
	if l.config == nil {
		return true
	}
	return convertLogLevel(level) >= convertLogLevel(l.config.MinLevel)
}

func (l *ZapLogger) Debug(msg string, fields ...types.LogField) {
	if l.shouldLog(types.LogLevelDebug) {
		l.logger.Debug(msg, convertToZapFields(fields)...)
	}
}

func (l *ZapLogger) Info(msg string, fields ...types.LogField) {
	if l.shouldLog(types.LogLevelInfo) {
		l.logger.Info(msg, convertToZapFields(fields)...)
	}
}

func (l *ZapLogger) Warn(msg string, fields ...types.LogField) {
	if l.shouldLog(types.LogLevelWarn) {
		l.logger.Warn(msg, convertToZapFields(fields)...)
	}
}

func (l *ZapLogger) Error(msg string, fields ...types.LogField) {
	if l.shouldLog(types.LogLevelError) {
		l.logger.Error(msg, convertToZapFields(fields)...)
	}
}

func (l *ZapLogger) With(fields ...types.LogField) types.Logger {
	return &ZapLogger{
		logger: l.logger.With(convertToZapFields(fields)...),
		config: l.config,
	}
}

func (l *ZapLogger) WithSampling(rate float64) types.Logger {
	if rate <= 0 || rate > 1 {
		return l
	}

	config := *l.config
	config.EnableSampling = true
	config.SampleRate = rate

	// Create a new logger with sampling enabled
	core := zapcore.NewSamplerWithOptions(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(convertLogLevel(config.MinLevel)),
	), time.Second, int(1/rate), int(1/rate))

	return &ZapLogger{
		logger: zap.New(core),
		config: &config,
	}
}

func (l *ZapLogger) Flush() error {
	return l.logger.Sync()
}

// ZapLoggerFactory creates ZapLogger instances
type ZapLoggerFactory struct {
	config     zap.Config
	baseFields []types.LogField
	logConfig  *types.LoggerConfig
}

// NewZapLoggerFactory creates a new ZapLoggerFactory
func NewZapLoggerFactory(config zap.Config) types.LoggerFactory {
	return &ZapLoggerFactory{
		config: config,
		logConfig: &types.LoggerConfig{
			MinLevel:       types.LogLevelInfo,
			EnableSampling: false,
		},
	}
}

// CreateLogger implements LoggerFactory.CreateLogger
func (f *ZapLoggerFactory) CreateLogger(name string) types.Logger {
	return f.CreateLoggerWithConfig(name, *f.logConfig)
}

// CreateLoggerWithConfig implements LoggerFactory.CreateLoggerWithConfig
func (f *ZapLoggerFactory) CreateLoggerWithConfig(name string, config types.LoggerConfig) types.Logger {
	zapConfig := f.config
	if zapConfig.Development {
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Apply config
	zapConfig.Level = zap.NewAtomicLevelAt(convertLogLevel(config.MinLevel))
	if config.EnableSampling {
		zapConfig.Sampling = &zap.SamplingConfig{
			Initial:    int(1 / config.SampleRate),
			Thereafter: int(1 / config.SampleRate),
		}
	} else {
		zapConfig.Sampling = nil
	}

	logger, err := zapConfig.Build(
		zap.AddCallerSkip(1),
		zap.Fields(zap.String("logger", name)),
	)
	if err != nil {
		logger, _ = zap.NewProduction()
	}

	// Add base fields
	fields := append(f.baseFields, config.DefaultFields...)
	if len(fields) > 0 {
		logger = logger.With(convertToZapFields(fields)...)
	}

	return NewZapLogger(logger, &config)
}

// WithFields implements LoggerFactory.WithFields
func (f *ZapLoggerFactory) WithFields(fields ...types.LogField) types.LoggerFactory {
	return &ZapLoggerFactory{
		config:     f.config,
		baseFields: append(f.baseFields, fields...),
		logConfig:  f.logConfig,
	}
}

// DefaultZapConfig returns a default Zap configuration
func DefaultZapConfig() zap.Config {
	return zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
}
