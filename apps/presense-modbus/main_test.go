package main

import (
	"encoding/binary"
	"testing"
)

func TestScaled(t *testing.T) {
	if got := scaled(23.56, 0.1); got != 236 {
		t.Fatalf("scaled() = %d, want 236", got)
	}
	if got := scaled(-1, 0.1); got != 0 {
		t.Fatalf("scaled negative = %d, want 0", got)
	}
}

func TestRegisterBankRead(t *testing.T) {
	bank := &RegisterBank{}
	bank.set([]uint16{11, 22, 33})
	values, ok := bank.read(1, 2)
	if !ok {
		t.Fatal("expected read to succeed")
	}
	if len(values) != 2 || values[0] != 22 || values[1] != 33 {
		t.Fatalf("unexpected values: %#v", values)
	}
	if _, ok := bank.read(99, 2); ok {
		t.Fatal("expected out-of-range read to fail")
	}
	if _, ok := bank.read(0, 0); ok {
		t.Fatal("expected zero-count read to fail")
	}
}

func TestExceptionResponse(t *testing.T) {
	response := exceptionResponse(123, 1, 3, 2)
	if len(response) != 9 {
		t.Fatalf("len(response) = %d, want 9", len(response))
	}
	if binary.BigEndian.Uint16(response[0:2]) != 123 {
		t.Fatalf("unexpected transaction id")
	}
	if response[6] != 1 || response[7] != 0x83 || response[8] != 2 {
		t.Fatalf("unexpected exception response: %#v", response)
	}
}

func TestEnvHelpers(t *testing.T) {
	t.Setenv("TEST_STRING", "value")
	if got := envString("TEST_STRING", "fallback"); got != "value" {
		t.Fatalf("envString = %q", got)
	}
	if got := envString("MISSING_STRING", "fallback"); got != "fallback" {
		t.Fatalf("envString fallback = %q", got)
	}

	t.Setenv("TEST_INT", "42")
	if got := envInt("TEST_INT", 1); got != 42 {
		t.Fatalf("envInt = %d", got)
	}
	t.Setenv("TEST_INT_BAD", "nope")
	if got := envInt("TEST_INT_BAD", 7); got != 7 {
		t.Fatalf("envInt bad fallback = %d", got)
	}

	t.Setenv("TEST_FLOAT", "4.25")
	if got := envFloat("TEST_FLOAT", 1.0); got != 4.25 {
		t.Fatalf("envFloat = %f", got)
	}
	t.Setenv("TEST_FLOAT_BAD", "nope")
	if got := envFloat("TEST_FLOAT_BAD", 8.5); got != 8.5 {
		t.Fatalf("envFloat bad fallback = %f", got)
	}
}
