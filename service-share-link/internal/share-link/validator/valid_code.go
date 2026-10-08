package validator

import (
	"errors"
	"service-share-link/internal/share-link/model"
	"service-share-link/internal/share-link/service"
	"strings"
)

func ValidateCode(code model.Code) error {
	if len(code) != service.LengthCode {
		return errors.New("code has invalid length")
	}

	for i := 0; i < len(code); i++ {
		if strings.IndexByte(service.AlphabetCode, code[i]) < 0 {
			return errors.New("code has invalid character")
		}
	}

	return nil
}
