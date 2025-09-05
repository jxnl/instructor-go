package core

const (
	DefaultMaxRetries = 3
	DefaultValidator  = false
)

type Options struct {
	Mode       *Mode
	MaxRetries *int
	Validate   *bool
	// Provider specific options:
}

var defaultOptions = Options{
	Mode:       ToPtr(ModeDefault),
	MaxRetries: ToPtr(DefaultMaxRetries),
	Validate:   ToPtr(DefaultValidator),
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

	return old
}

func MergeOptions(opts ...Options) Options {
	options := defaultOptions

	for _, opt := range opts {
		options = mergeOption(options, opt)
	}

	return options
}
