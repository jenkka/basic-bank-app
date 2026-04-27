package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSupportedCurrency(t *testing.T) {
	for _, currency := range SupportedCurrencies {
		t.Run(currency, func(t *testing.T) {
			require.True(t, IsSupportedCurrency(currency))
		})
	}

	for _, currency := range []string{"", "RUB", "usd", "123"} {
		t.Run(currency, func(t *testing.T) {
			require.False(t, IsSupportedCurrency(currency))
		})
	}
}
