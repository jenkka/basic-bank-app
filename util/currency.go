package util

import "slices"

const (
	USD = "USD"
	EUR = "EUR"
	CAD = "CAD"
	MXN = "MXN"
	GBP = "GBP"
	JPY = "JPY"
	CHF = "CHF"
	AUD = "AUD"
	BRL = "BRL"
	INR = "INR"
)

var SupportedCurrencies = []string{
	USD, EUR, CAD, MXN, GBP, JPY, CHF, AUD, BRL, INR,
}

func IsSupportedCurrency(currency string) bool {
	return slices.Contains(SupportedCurrencies, currency)
}
