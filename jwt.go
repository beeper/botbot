package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util/jsontime"
)

type jwtPayload struct {
	Subject   string        `json:"sub"`
	Issuer    string        `json:"iss"`
	IssuedAt  jsontime.Unix `json:"iat"`
	ExpiresAt jsontime.Unix `json:"exp"`
}

// var encodedJWTHeader = base64.RawStdEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
const encodedJWTHeader = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`

func createLoginToken(userID id.UserID) string {
	payload, err := json.Marshal(&jwtPayload{
		Subject:   userID.Localpart(),
		Issuer:    "botbot",
		IssuedAt:  jsontime.UnixNow(),
		ExpiresAt: jsontime.U(time.Now().Add(1 * time.Minute)),
	})
	if err != nil {
		panic(fmt.Errorf("failed to marshal JWT payload: %w", err))
	}

	signer := hmac.New(sha256.New, []byte(cfg.LoginJWTKey))

	headerEnd := len(encodedJWTHeader)
	dataStart := headerEnd + 1
	dataEnd := dataStart + base64.RawStdEncoding.EncodedLen(len(payload))
	signatureStart := dataEnd + 1
	signatureEnd := signatureStart + base64.RawStdEncoding.EncodedLen(signer.Size())

	encodedJWT := make([]byte, signatureEnd)

	copy(encodedJWT[:headerEnd], encodedJWTHeader)
	encodedJWT[headerEnd] = '.'
	encodedJWT[dataEnd] = '.'

	base64.RawStdEncoding.Encode(encodedJWT[dataStart:dataEnd], payload)
	signer.Write(encodedJWT[:dataEnd])
	base64.RawStdEncoding.Encode(encodedJWT[signatureStart:], signer.Sum(nil))

	return string(encodedJWT)
}
