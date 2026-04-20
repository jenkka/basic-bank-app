package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPassword(t *testing.T) {
	// Check against the correct password
	password := RandomString(8)
	hashedPassword1, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword1)
	require.True(t, CheckPassword(password, hashedPassword1))

	// Test that two hashed passwords generated from the same original password
	// are different
	hashedPassword2, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword2)
	require.NotEqual(t, hashedPassword1, hashedPassword2)

	// Check against a wrong password
	wrongPassword := RandomString(10)
	require.False(t, CheckPassword(wrongPassword, hashedPassword1))
}
