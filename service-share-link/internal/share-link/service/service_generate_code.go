package service

import (
	"crypto/rand"
	"service-share-link/internal/share-link/model"
)

const (
	LengthCode   = 8
	AlphabetCode = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func GeneratorCode() (model.Code, error) {
	code := make([]byte, 0, LengthCode)
	buf := make([]byte, LengthCode)

	for len(code) < LengthCode {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if b >= 248 {
				continue
			}

			code = append(code, AlphabetCode[b%62])
			if len(code) == LengthCode {
				break
			}
		}
	}
	return model.Code(code), nil
}
