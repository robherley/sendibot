package sendico

import (
	"errors"
	"fmt"
)

var (
	ErrRequest        = errors.New("sendico request failed")
	ErrSecretNotFound = errors.New("sendico API secret not found")
	ErrInvalidShop    = errors.New("invalid shop")
)

func NewRequestError(err error) error {
	return fmt.Errorf("%w: %w", ErrRequest, err)
}

func NewUnexpectedResponseCodeError(code int) error {
	return fmt.Errorf("%w: %w", ErrRequest, fmt.Errorf("unexpected status code: %d", code))
}

func NewInvalidShopError(s string) error {
	return fmt.Errorf("%w: %q", ErrInvalidShop, s)
}
