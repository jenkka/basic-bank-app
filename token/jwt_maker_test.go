package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jenkka/dummy-bank/util"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)
	require.NotNil(t, maker)

	username := util.RandomUsername()
	duration := time.Minute
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(duration)

	token, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	payload, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username)
	require.WithinDuration(t, issuedAt, payload.IssuedAt.Time, time.Second)
	require.WithinDuration(t, expiresAt, payload.ExpiresAt.Time, time.Second)
}

func TestExpiredJWTToken(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)
	require.NotNil(t, maker)

	username := util.RandomUsername()
	duration := -time.Minute

	token, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	payload, err := maker.VerifyToken(token)
	require.Nil(t, payload)
	require.ErrorIs(t, err, ErrExpiredToken)
}

func TestInvalidJWTTokenAlgNone(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)
	require.NotNil(t, maker)

	username := util.RandomUsername()
	duration := time.Minute
	payload, err := NewPayload(username, duration)
	require.NoError(t, err)
	require.NotNil(t, payload)

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, payload)
	token, _ := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)

	payload, err = maker.VerifyToken(token)
	require.Nil(t, payload)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestInvalidJWTTokenBadSignature(t *testing.T) {
	makerA, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	makerB, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	token, err := makerA.CreateToken(util.RandomUsername(), time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	payload, err := makerB.VerifyToken(token)
	require.Nil(t, payload)
	require.ErrorIs(t, err, ErrInvalidToken)
}
