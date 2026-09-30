package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var v = func() *validator.Validate {
	vld := validator.New()

	// Use JSON field names in error messages instead of struct field names.
	vld.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return vld
}()

// ValidationError holds a list of field-level errors.
type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		msgs[i] = f.Field + ": " + f.Message
	}
	return strings.Join(msgs, "; ")
}

// Engine returns the underlying go-playground validator (tag "validate").
// Used to plug this engine into gin's binding layer.
func Engine() *validator.Validate { return v }

// Validate runs go-playground/validator on v and returns a *ValidationError
// if any field constraints are violated, or nil otherwise.
func Validate(input any) error {
	err := v.Struct(input)
	if err == nil {
		return nil
	}

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return err
	}

	fields := make([]FieldError, 0, len(ve))
	for _, e := range ve {
		fields = append(fields, FieldError{
			Field:   e.Field(),
			Message: fieldMessage(e),
		})
	}
	return &ValidationError{Fields: fields}
}

func fieldMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "field is required"
	case "email":
		return "invalid email format"
	case "min":
		return "must be at least " + e.Param() + " characters"
	case "max":
		return "must be at most " + e.Param() + " characters"
	case "len":
		return "must be exactly " + e.Param() + " characters"
	case "uuid", "uuid4":
		return "invalid UUID format"
	case "oneof":
		return "must be one of: " + e.Param()
	case "gt":
		return "must be greater than " + e.Param()
	case "gte":
		return "must be greater than or equal to " + e.Param()
	case "lt":
		return "must be less than " + e.Param()
	case "lte":
		return "must be less than or equal to " + e.Param()
	case "httpurl":
		return "must be a valid http(s) URL"
	case "url":
		return "invalid URL format"
	default:
		return "invalid value (" + e.Tag() + ")"
	}
}
