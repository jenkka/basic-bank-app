package util

import (
	"fmt"
	"math/rand"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

func RandomUsername() string {
	return RandomString(6)
}

func RandomMoney() int64 {
	return RandomInt(1, 1000)
}

func RandomCurrency() string {
	currencies := []string{USD, EUR, CAD, MXN}
	n := len(currencies)
	return currencies[rand.Intn(n)]
}

func RandomEmailDomain() string {
	domains := []string{
		"gmail.com",
		"outlook.com",
		"proton.me",
		"hotmail.com",
	}

	return domains[rand.Intn(len(domains))]
}

func RandomEmail() string {
	return fmt.Sprintf("%s@%s", RandomString(8), RandomEmailDomain())
}

func RandomFullName() string {
	return fmt.Sprintf("%s %s %s", RandomString(6), RandomString(6), RandomString(8))
}
