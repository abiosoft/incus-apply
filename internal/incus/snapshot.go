package incus

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// snapshotEncoder encodes a YAML snapshot string for storage in a resource config key.
type snapshotEncoder interface {
	version() string
	Encode(snapshotYAML string) (string, error)
}

// snapshotDecoder decodes a stored snapshot value back to a YAML string.
type snapshotDecoder interface {
	Decode(encoded string) (string, error)
}

// snapshotCodec implements both snapshotEncoder and snapshotDecoder.
type snapshotCodec interface {
	snapshotEncoder
	snapshotDecoder
}

var _ snapshotCodec = v1SnapshotCodec{}
var _ snapshotCodec = legacySnapshotCodec{}

// v1SnapshotCodec implements snapshotCodec for version "1":
// YAML → gzip → base64 on encode; base64 → gunzip on decode.
type v1SnapshotCodec struct{}

func (v1SnapshotCodec) version() string { return snapshotVersion }

func (v1SnapshotCodec) Encode(snapshotYAML string) (string, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte(snapshotYAML)); err != nil {
		return "", fmt.Errorf("compressing managed state: %w", err)
	}
	if err := gw.Close(); err != nil {
		return "", fmt.Errorf("compressing managed state: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (v1SnapshotCodec) Decode(encoded string) (string, error) {
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decoding snapshot: %w", err)
	}
	gr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", fmt.Errorf("creating gzip reader for snapshot: %w", err)
	}
	defer func() { _ = gr.Close() }()
	data, err := io.ReadAll(gr)
	if err != nil {
		return "", fmt.Errorf("decompressing snapshot: %w", err)
	}
	return string(data), nil
}

// legacySnapshotCodec handles pre-versioned snapshots stored as plain YAML/JSON.
// Encode is a no-op: legacy snapshots are stored as-is without transformation.
type legacySnapshotCodec struct{}

func (legacySnapshotCodec) version() string                            { return "" }
func (legacySnapshotCodec) Encode(snapshotYAML string) (string, error) { return snapshotYAML, nil }
func (legacySnapshotCodec) Decode(encoded string) (string, error)      { return encoded, nil }

// codecForVersion returns the snapshotCodec for the given snapshot_version value.
// An empty version string selects the legacy codec.
// The second return value is false if the version is unrecognised.
func codecForVersion(version string) (snapshotCodec, bool) {
	switch strings.TrimSpace(version) {
	case "":
		return legacySnapshotCodec{}, true
	case snapshotVersion:
		return v1SnapshotCodec{}, true
	default:
		return nil, false
	}
}
