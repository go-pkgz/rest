package rest

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/pkg/errors"
)

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// RenderJSONFromBytes sends binary data as json
func RenderJSONFromBytes(w http.ResponseWriter, r *http.Request, data []byte) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}
	if _, err := w.Write(data); err != nil {
		return errors.Wrapf(err, "failed to send response to %s", r.RemoteAddr)
	}
	return nil
}

// RenderJSONWithHTML allows html tags and forces charset=utf-8
func RenderJSONWithHTML(w http.ResponseWriter, r *http.Request, v interface{}) {
	data, err := encodeJSONWithHTML(v)
	if err != nil {
		SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't render json response")
		return
	}
	RenderJSONFromBytes(w, r, data)
}

func encodeJSONWithHTML(v interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, errors.Wrap(err, "json encoding failed")
	}
	return buf.Bytes(), nil
}
