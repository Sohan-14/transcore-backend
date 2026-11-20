package config

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"
)

func asString(s string) (string, error) {
	return s, nil
}

func asBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}

func asInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func asInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func asDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

func asLevel(s string) (slog.Level, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return 0, err
	}
	return l, nil
}

func oneOf[T ~string](allowed ...T) func(string) (T, error) {
	return func(s string) (T, error) {
		v := T(s)
		if !slices.Contains(allowed, v) {
			var zero T
			return zero, fmt.Errorf("%q is not allowed, want one of %v", s, allowed)
		}
		return v, nil
	}
}
