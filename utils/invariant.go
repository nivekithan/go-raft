package utils

import "log/slog"

func Invariant(l *slog.Logger, isPassed bool, message string) {

	if l == nil {
		l = slog.Default()
	}
	if !isPassed {
		l.Error(message)
	}
}
