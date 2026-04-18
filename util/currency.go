package util

const (
	USD = "USD"
	EUR = "EUR"
	CAD = "CAD"
	MXN = "MXN"
)

func IsSupportedCurrency(currency string) bool {
	switch currency {
	case USD, EUR, CAD, MXN:
		return true
	}

	return false
}
