package rotator

import (
	"github.com/natefinch/lumberjack"
)

// Options for a Rotator that rotates log files. A zero Options consists entirely
// of default values.
type Options struct {
	Filename   string
	MaxSize    int
	MaxAge     int
	MaxBackups int
	LocalTime  bool
	Compress   bool
}

// A Rotator represents an active rotating object that uses lumberjack.Logger
// to rotate log files.
type Rotator struct {
	Logger *lumberjack.Logger
}

// New creates a new [Rotator].
func New(opts *Options) *Rotator {
	var r Rotator

	r.Logger = &lumberjack.Logger{
		Filename: ".app/logs/logs.log",
		MaxAge:   28,
		Compress: true,
	}

	if opts == nil {
		return &r
	}

	if opts.Filename != "" {
		r.Logger.Filename = opts.Filename
	}

	if opts.MaxSize != 0 {
		r.Logger.MaxSize = opts.MaxSize
	}

	if opts.MaxBackups != 0 {
		r.Logger.MaxBackups = opts.MaxBackups
	}

	if opts.MaxAge != 0 {
		r.Logger.MaxAge = opts.MaxAge
	}

	if opts.LocalTime {
		r.Logger.LocalTime = true
	}

	if opts.Compress {
		r.Logger.Compress = true
	}

	return &r
}
