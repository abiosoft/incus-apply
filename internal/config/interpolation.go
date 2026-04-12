package config

import (
	"fmt"
	"strings"
)

// Interpolate replaces variable references in data with values from env.
//
// Supported syntax:
//
//	$$               literal '$'
//	$VAR             value of VAR (empty string if not set)
//	${VAR}           value of VAR (empty string if not set)
//	${VAR:-default}  value of VAR if set and non-empty, otherwise default
func Interpolate(data []byte, env map[string]string) ([]byte, error) {
	return interpolate(data, env, false)
}

// InterpolateStrict is like Interpolate but returns an error for undeclared variables.
func InterpolateStrict(data []byte, env map[string]string) ([]byte, error) {
	return interpolate(data, env, true)
}

// InterpolateDeclared replaces only declared variable references in data.
// Undeclared variables are preserved as-is.
func InterpolateDeclared(data []byte, env map[string]string) ([]byte, error) {
	return interpolateDeclared(data, env)
}

func interpolate(data []byte, env map[string]string, strict bool) ([]byte, error) {
	s := string(data)
	var buf strings.Builder
	buf.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] != '$' {
			buf.WriteByte(s[i])
			i++
			continue
		}
		if i+1 >= len(s) {
			buf.WriteByte('$')
			i++
			continue
		}
		switch s[i+1] {
		case '$':
			buf.WriteByte('$')
			i += 2
		case '{':
			end := strings.Index(s[i+2:], "}")
			if end < 0 {
				return nil, fmt.Errorf("variable expression missing closing '}' near position %d", i)
			}
			expr := s[i+2 : i+2+end]
			val, err := resolveExpr(expr, env, strict)
			if err != nil {
				return nil, err
			}
			buf.WriteString(val)
			i = i + 2 + end + 1
		default:
			if isVarStart(s[i+1]) {
				j := i + 1
				for j < len(s) && isVarChar(s[j]) {
					j++
				}
				name := s[i+1 : j]
				val, ok := env[name]
				if strict && !ok {
					return nil, fmt.Errorf("undeclared variable %q", name)
				}
				buf.WriteString(val)
				i = j
			} else {
				buf.WriteByte('$')
				i++
			}
		}
	}
	return []byte(buf.String()), nil
}

func interpolateDeclared(data []byte, env map[string]string) ([]byte, error) {
	s := string(data)
	var buf strings.Builder
	buf.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] != '$' {
			buf.WriteByte(s[i])
			i++
			continue
		}
		if i+1 >= len(s) {
			buf.WriteByte('$')
			i++
			continue
		}
		switch s[i+1] {
		case '$':
			buf.WriteByte('$')
			i += 2
		case '{':
			end := strings.Index(s[i+2:], "}")
			if end < 0 {
				return nil, fmt.Errorf("variable expression missing closing '}' near position %d", i)
			}
			raw := s[i : i+2+end+1]
			expr := s[i+2 : i+2+end]
			val, replaced, err := resolveDeclaredExpr(expr, env)
			if err != nil {
				return nil, err
			}
			if replaced {
				buf.WriteString(val)
			} else {
				buf.WriteString(raw)
			}
			i = i + 2 + end + 1
		default:
			if isVarStart(s[i+1]) {
				j := i + 1
				for j < len(s) && isVarChar(s[j]) {
					j++
				}
				name := s[i+1 : j]
				if val, ok := env[name]; ok {
					buf.WriteString(val)
				} else {
					buf.WriteString(s[i:j])
				}
				i = j
			} else {
				buf.WriteByte('$')
				i++
			}
		}
	}
	return []byte(buf.String()), nil
}

// resolveExpr evaluates the contents of a ${...} expression.
//
// Supported forms:
//
//	${VAR}           value of VAR
//	${VAR:-default}  value of VAR if set and non-empty, otherwise default
func resolveExpr(expr string, env map[string]string, strict bool) (string, error) {
	if len(expr) == 0 || !isVarStart(expr[0]) {
		return "", fmt.Errorf("invalid variable expression: %q", expr)
	}

	// Extract variable name
	i := 0
	for i < len(expr) && isVarChar(expr[i]) {
		i++
	}
	name := expr[:i]

	// Plain ${VAR}
	if i == len(expr) {
		val, ok := env[name]
		if strict && !ok {
			return "", fmt.Errorf("undeclared variable %q", name)
		}
		return val, nil
	}

	rest := expr[i:]

	// ${VAR:-default}
	if strings.HasPrefix(rest, ":-") {
		def := rest[2:]
		val, ok := env[name]
		if strict && !ok {
			return def, nil // default provided, so not an error
		}
		if ok && val != "" {
			return val, nil
		}
		return def, nil
	}

	return "", fmt.Errorf("unsupported variable expression: ${%s}", expr)
}

func resolveDeclaredExpr(expr string, env map[string]string) (string, bool, error) {
	if len(expr) == 0 || !isVarStart(expr[0]) {
		return "", false, fmt.Errorf("invalid variable expression: %q", expr)
	}

	i := 0
	for i < len(expr) && isVarChar(expr[i]) {
		i++
	}
	name := expr[:i]

	if i == len(expr) {
		val, ok := env[name]
		if !ok {
			return "", false, nil
		}
		return val, true, nil
	}

	rest := expr[i:]
	if strings.HasPrefix(rest, ":-") {
		val, ok := env[name]
		if !ok {
			return "", false, nil
		}
		if val != "" {
			return val, true, nil
		}
		return rest[2:], true, nil
	}

	return "", false, fmt.Errorf("unsupported variable expression: ${%s}", expr)
}

func isVarStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isVarChar(c byte) bool {
	return isVarStart(c) || (c >= '0' && c <= '9')
}
