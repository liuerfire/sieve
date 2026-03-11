package retry

func WithRetry[T any](retries int, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	for i := range retries + 1 {
		value, err := fn()
		if err == nil {
			return value, nil
		}
		lastErr = err
		if i == retries {
			break
		}
	}
	return zero, lastErr
}
