package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is required")
	}
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("malformed JSON at position %d", syntaxErr.Offset)
		case errors.As(err, &unmarshalErr):
			return fmt.Errorf("invalid value for field %q", unmarshalErr.Field)
		case errors.Is(err, io.EOF):
			return fmt.Errorf("request body is empty")
		default:
			return err
		}
	}

	return nil
}