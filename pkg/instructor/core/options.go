package core

const (
	DefaultMaxRetries = 3
	DefaultValidator  = false
)

type Options struct {
	Mode       *Mode
	MaxRetries *int
	Validate   *bool
	Logger     Logger
	// Provider specific options:
}

var defaultOptions = Options{
	Mode:       ToPtr(ModeDefault),
	MaxRetries: ToPtr(DefaultMaxRetries),
	Validate:   ToPtr(DefaultValidator),
	Logger:     NewNoopLogger(), // Silent by default
}

func WithMode(mode Mode) Options {
	return Options{Mode: ToPtr(mode)}
}

func WithMaxRetries(maxRetries int) Options {
	return Options{MaxRetries: ToPtr(maxRetries)}
}

func WithValidation() Options {
	return Options{Validate: ToPtr(true)}
}

func WithLogger(logger Logger) Options {
	return Options{Logger: logger}
}

// WithLogging is a convenience function for common logging setups
// Examples:
//   - WithLogging("debug") - Text logging at DEBUG level to stderr
//   - WithLogging("info") - Text logging at INFO level to stderr
//   - WithLogging("warn") - Text logging at WARN level to stderr
//   - WithLogging("error") - Text logging at ERROR level to stderr
//   - WithLogging("json") - JSON logging at INFO level to stderr
//   - WithLogging("json:debug") - JSON logging at DEBUG level to stderr
func WithLogging(level string) Options {
	logger := NewLoggerFromString(level)
	return Options{Logger: logger}
}

func mergeOption(old, new Options) Options {
	if new.Mode != nil {
		old.Mode = new.Mode
	}
	if new.MaxRetries != nil {
		old.MaxRetries = new.MaxRetries
	}
	if new.Validate != nil {
		old.Validate = new.Validate
	}
	if new.Logger != nil {
		old.Logger = new.Logger
	}

	return old
}

func MergeOptions(opts ...Options) Options {
	options := defaultOptions

	for _, opt := range opts {
		options = mergeOption(options, opt)
	}

	return options
}
