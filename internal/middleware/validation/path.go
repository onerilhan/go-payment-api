package validation

import (
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Pre-compiled patterns for performance
var (
	uuidPattern         = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	alphanumericPattern = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	slugPattern         = regexp.MustCompile(`^[a-z0-9\-]+$`)
	hexPattern          = regexp.MustCompile(`^[0-9a-fA-F]+$`)
)

// ValidatePathParameters path parameters validation
func ValidatePathParameters(r *http.Request, pathValidation map[string]string) error {
	vars := mux.Vars(r)

	for paramName, validationRule := range pathValidation {
		paramValue, exists := vars[paramName]
		if !exists {
			continue // Parameter yoksa skip
		}

		// Boş parametre kontrolü
		if paramValue == "" {
			return fmt.Errorf("path parameter '%s' boş olamaz", paramName)
		}

		switch validationRule {
		case "integer":
			if err := validateInteger(paramName, paramValue); err != nil {
				return err
			}
		case "positive_integer":
			if err := validatePositiveInteger(paramName, paramValue); err != nil {
				return err
			}
		case "uuid":
			if err := validateUUID(paramName, paramValue); err != nil {
				return err
			}
		case "alphanumeric":
			if err := validateAlphanumeric(paramName, paramValue); err != nil {
				return err
			}
		case "slug":
			if err := validateSlug(paramName, paramValue); err != nil {
				return err
			}
		case "hex":
			if err := validateHex(paramName, paramValue); err != nil {
				return err
			}
		case "date":
			if err := validateDate(paramName, paramValue); err != nil {
				return err
			}
		case "email":
			if err := validateEmail(paramName, paramValue); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateInteger integer validation
func validateInteger(paramName, paramValue string) error {
	if _, err := strconv.Atoi(paramValue); err != nil {
		return fmt.Errorf("path parameter '%s' integer olmalı, alınan: '%s'", paramName, paramValue)
	}
	return nil
}

// validatePositiveInteger positive integer validation
func validatePositiveInteger(paramName, paramValue string) error {
	val, err := strconv.Atoi(paramValue)
	if err != nil || val <= 0 {
		return fmt.Errorf("path parameter '%s' pozitif integer olmalı, alınan: '%s'", paramName, paramValue)
	}
	return nil
}

// validateUUID UUID format validation
func validateUUID(paramName, paramValue string) error {
	if !uuidPattern.MatchString(paramValue) {
		return fmt.Errorf("path parameter '%s' geçerli UUID formatında olmalı, alınan: '%s'", paramName, paramValue)
	}
	return nil
}

// validateAlphanumeric alphanumeric validation
func validateAlphanumeric(paramName, paramValue string) error {
	if !alphanumericPattern.MatchString(paramValue) {
		return fmt.Errorf("path parameter '%s' sadece harf ve rakam içerebilir, alınan: '%s'", paramName, paramValue)
	}
	return nil
}

// validateSlug URL-friendly slug validation
func validateSlug(paramName, paramValue string) error {
	if !slugPattern.MatchString(paramValue) {
		return fmt.Errorf("path parameter '%s' geçerli slug formatında olmalı (a-z, 0-9, -), alınan: '%s'", paramName, paramValue)
	}
	return nil
}

// validateHex hexadecimal validation
func validateHex(paramName, paramValue string) error {
	if !hexPattern.MatchString(paramValue) {
		return fmt.Errorf("path parameter '%s' hexadecimal formatında olmalı, alınan: '%s'", paramName, paramValue)
	}
	return nil
}

// validateDate date validation with multiple formats
func validateDate(paramName, paramValue string) error {
	// Supported date formats
	dateFormats := []string{
		"2006-01-02", // YYYY-MM-DD (ISO)
		"2006/01/02", // YYYY/MM/DD
		"02-01-2006", // DD-MM-YYYY
		"02/01/2006", // DD/MM/YYYY
	}

	for _, format := range dateFormats {
		if _, err := time.Parse(format, paramValue); err == nil {
			return nil // Valid format found
		}
	}

	return fmt.Errorf(
		"path parameter '%s' geçerli tarih formatında olmalı (YYYY-MM-DD, YYYY/MM/DD, DD-MM-YYYY, DD/MM/YYYY), alınan: '%s'",
		paramName, paramValue,
	)
}

// validateEmail improved email validation using net/mail
func validateEmail(paramName, paramValue string) error {
	_, err := mail.ParseAddress(paramValue)
	if err != nil {
		return fmt.Errorf("path parameter '%s' geçerli email formatında olmalı, alınan: '%s'", paramName, paramValue)
	}
	return nil
}
