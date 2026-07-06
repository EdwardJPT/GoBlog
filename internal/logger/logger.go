package logger

import (
	"log/slog"
	"os"
)

// Init configures the global slog instance with JSON formatting
// and strictly fixed-length microsecond timestamps.
func Init() {
	// 6 zeros force exactly 6 digits for microseconds
	const fixedMicroLayout = "2006-01-02T15:04:05.000000Z07:00"

	opts := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Target the default time key at the root level
			if a.Key == slog.TimeKey && len(groups) == 0 {
				t := a.Value.Time()
				return slog.String(slog.TimeKey, t.Format(fixedMicroLayout))
			}
			return a
		},
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(jsonHandler)

	// Set this as the global default logger
	slog.SetDefault(logger)
}
