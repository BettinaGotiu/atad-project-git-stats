package main

import (
	"atad-project/internal/stats"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("expected one JSON object")
	}
	return nil
}

func (s service) identities(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Rules *[]stats.Rule `json:"rules"`
	}
	if err := decode(w, r, &data); err != nil {
		problem(w, 400, "validation", err.Error())
		return
	}
	if data.Rules == nil {
		problem(w, 400, "validation", "rules must be an array")
		return
	}
	aliases, err := stats.NewAliases(*data.Rules)
	if err != nil {
		problem(w, 400, "validation", err.Error())
		return
	}
	s.jobs.SetAliases(aliases)
	w.WriteHeader(http.StatusNoContent)
}
