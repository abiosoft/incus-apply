package config

import (
	"testing"
)

func TestInterpolate(t *testing.T) {
	env := map[string]string{"NAME": "world", "EMPTY": "", "incus.remote.get-client-certificate": "FAKECERT", "incus.remote.get-client-certificate-base64": "RkFLRUNFUlQ="}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Basic
		{"no vars", "hello", "hello", false},
		{"unbraced", "hello $NAME", "hello world", false},
		{"braced", "hello ${NAME}", "hello world", false},
		{"escaped dollar", "price $$1", "price $1", false},
		{"unset resolves empty", "$UNSET", "", false},
		{"adjacent vars", "${NAME}${NAME}", "worldworld", false},
		{"dollar at end of string", "foo$", "foo$", false},
		{"dollar before non-var char", "foo$ bar", "foo$ bar", false},
		{"empty braced is ok", "${EMPTY}", "", false},

		// Default values
		{":-  unset uses default", "${UNSET:-fallback}", "fallback", false},
		{":-  empty uses default", "${EMPTY:-fallback}", "fallback", false},
		{":-  set returns value", "${NAME:-fallback}", "world", false},

		// Errors
		{"unclosed brace", "${NAME", "", true},
		{"invalid name in braces", "${123}", "", true},

		// incus.* variables
		{"incus dotted var", "${incus.remote.get-client-certificate}", "FAKECERT", false},
		{"incus dotted var with default uses value", "${incus.remote.get-client-certificate:-default}", "FAKECERT", false},
		{"incus dotted unset uses default", "${incus.other.var:-fallback}", "fallback", false},
		{"incus dotted unset resolves empty", "${incus.other.var}", "", false},
		{"incus dotted base64 var", "${incus.remote.get-client-certificate-base64}", "RkFLRUNFUlQ=", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Interpolate([]byte(tt.input), env)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && string(got) != tt.want {
				t.Errorf("got %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestInterpolateStrict(t *testing.T) {
	env := map[string]string{"NAME": "world"}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"declared var", "$NAME", "world", false},
		{"undeclared braced", "${UNSET}", "", true},
		{"undeclared unbraced", "$UNSET", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InterpolateStrict([]byte(tt.input), env)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && string(got) != tt.want {
				t.Errorf("got %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestInterpolateDeclared(t *testing.T) {
	env := map[string]string{"NAME": "world", "EMPTY": "", "incus.remote.get-client-certificate": "FAKECERT", "incus.remote.get-client-certificate-base64": "RkFLRUNFUlQ="}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"declared unbraced", "hello $NAME", "hello world", false},
		{"undeclared unbraced preserved", "hello $UNSET", "hello $UNSET", false},
		{"declared braced", "hello ${NAME}", "hello world", false},
		{"undeclared braced preserved", "hello ${UNSET}", "hello ${UNSET}", false},
		{"declared default uses default for empty", "${EMPTY:-fallback}", "fallback", false},
		{"undeclared default preserved", "${UNSET:-fallback}", "${UNSET:-fallback}", false},
		{"escaped dollar", "price $$1", "price $1", false},
		{"unclosed brace", "${NAME", "", true},
		{"invalid name in braces", "${123}", "", true},

		// incus.* variables
		{"incus declared resolved", "${incus.remote.get-client-certificate}", "FAKECERT", false},
		{"incus undeclared preserved", "${incus.unknown.var}", "${incus.unknown.var}", false},
		{"incus undeclared default preserved", "${incus.unknown.var:-fallback}", "${incus.unknown.var:-fallback}", false},
		{"incus declared default uses value", "${incus.remote.get-client-certificate:-fallback}", "FAKECERT", false},
		{"incus declared base64 resolved", "${incus.remote.get-client-certificate-base64}", "RkFLRUNFUlQ=", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InterpolateDeclared([]byte(tt.input), env)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && string(got) != tt.want {
				t.Errorf("got %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestInterpolate_multiline(t *testing.T) {
	env := map[string]string{"DB": "mysql", "PASS": "secret"}
	input := "host: ${DB}\npassword: ${PASS}\n"
	want := "host: mysql\npassword: secret\n"

	got, err := Interpolate([]byte(input), env)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", string(got), want)
	}
}
