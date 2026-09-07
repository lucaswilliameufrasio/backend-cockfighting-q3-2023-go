package main

import "testing"

func TestIsDateValidValid(t *testing.T) {
	valid := []string{
		"1990-01-01",
		"2024-02-29",
		"2000-02-29",
		"1800-01-01",
		"9999-12-31",
	}
	for _, d := range valid {
		if !isDateValid(d) {
			t.Errorf("esperava válido: %q", d)
		}
	}
}

func TestIsDateValidInvalid(t *testing.T) {
	invalid := []string{
		"",
		"not-a-date",
		"1990-13-01",
		"1990-00-01",
		"1990-01-32",
		"2023-02-29",
		"1900-02-29",
		"1990-04-31",
		"1799-01-01",
		"10000-01-01",
		"1990/01/01",
		"99-01-01",
		"1990-1-1",
		"1990-01-0a",
		" 90-01-01",
	}
	for _, d := range invalid {
		if isDateValid(d) {
			t.Errorf("esperava inválido: %q", d)
		}
	}
}
