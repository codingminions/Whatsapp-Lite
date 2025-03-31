package validator

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator is an interface for validating structs
type Validator interface {
	// Validate checks the input against validation rules
	Validate(i interface{}) error
}

// CustomValidator implements Validator using the go-playground/validator package
type CustomValidator struct {
	validator *validator.Validate
}

// NewCustomValidator creates a new validator
func NewCustomValidator() *CustomValidator {
	v := validator.New()

	// Register a custom function to get JSON tag names
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &CustomValidator{
		validator: v,
	}
}

// Validate validates a struct
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			messages := make([]string, 0, len(validationErrors))
			for _, e := range validationErrors {
				messages = append(messages, formatValidationError(e))
			}
			return errors.New(strings.Join(messages, "; "))
		}
		return err
	}
	return nil
}

// formatValidationError formats a validation error
func formatValidationError(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must not be longer than %s characters", field, e.Param())
	default:
		return fmt.Sprintf("%s failed validation: %s", field, e.Tag())
	}
}
