package payload

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type Data map[string]any

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, data Data, headers http.Header) error {
	res, err := json.Marshal(data)

	if err != nil {
		return err
	}

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(res)
	return nil
}

func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	maxBytes := 1048756 // 1 MB
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(dst)

	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains bad JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains bad JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body cant be empty")

		case strings.HasPrefix(err.Error(), "json: Unkown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: Unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must be less than %d bytes", maxBytesError.Limit)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	return nil
}

func Validate(dst any) []ValidationError {
	validate := validator.New()

	if err := validate.Struct(dst); err != nil {
		ve := err.(validator.ValidationErrors)
		out := make([]ValidationError, len(ve))

		for i, v := range ve {
			out[i] = ValidationError{Field: v.Field(), Message: msgForTag(v)}
		}

		return out
	}

	return nil
}

func msgForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "min":
		return fmt.Sprintf("This field must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("This field must be at most %s", fe.Param())
	}

	return fe.Error()
}

func QueryInt(r *http.Request, key string) (int64, error) {
	value := r.URL.Query().Get(key)

	if value == "" {
		return 0, errors.New("empty query param")
	}

	query, err := strconv.ParseInt(value, 10, 64)

	if err != nil {
		return 0, errors.New("invalid query param")
	}

	return query, nil
}

func ParamInt(r *http.Request, key string) (int64, error) {
	paramStr := chi.URLParam(r, key)

	if paramStr == "" {
		return 0, errors.New("empty param")
	}

	param, err := strconv.ParseInt(paramStr, 10, 64)

	if err != nil {
		return 0, errors.New("invalid param")
	}

	return param, nil
}
