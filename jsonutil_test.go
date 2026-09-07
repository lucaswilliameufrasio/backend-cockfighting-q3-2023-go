package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONResponseSetsContentLength(t *testing.T) {
	rec := httptest.NewRecorder()
	jsonResponse(rec, http.StatusOK, map[string]string{"id": "abc"})

	if cl := rec.Header().Get("Content-Length"); cl == "" {
		t.Fatal("Content-Length ausente: cairia em chunked encoding")
	}
	if te := rec.Header().Get("Transfer-Encoding"); te != "" {
		t.Fatalf("Transfer-Encoding não deveria estar presente: %q", te)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type inesperado: %q", ct)
	}
}

func TestErrorResponseIsJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	errorResponse(rec, http.StatusUnprocessableEntity, "Conflict")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status inesperado: %d", rec.Code)
	}
	if cl := rec.Header().Get("Content-Length"); cl == "" {
		t.Fatal("Content-Length ausente em errorResponse")
	}
}
