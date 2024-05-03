package xtrace

import "github.com/solarwinds/apm-go/internal/log"

type AuthStatus int

const (
	AuthOK = iota
	AuthBadTimestamp
	AuthNoSignatureKey
	AuthBadSignature
)

func (a AuthStatus) IsError() bool {
	return a != AuthOK
}

func (a AuthStatus) Msg() string {
	switch a {
	case AuthOK:
		return "ok"
	case AuthBadTimestamp:
		return "bad-timestamp"
	case AuthNoSignatureKey:
		return "no-signature-key"
	case AuthBadSignature:
		return "bad-signature"
	}
	log.Debugf("could not read msg for unknown AuthStatus: %s", a)
	return ""
}
