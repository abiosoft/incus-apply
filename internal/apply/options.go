package apply

import "time"

// Options holds all CLI flags and configuration options.
type Options struct {
	// Input sources (positional arguments: files, directories, URLs, or '-' for stdin)
	Files     []string
	Recursive bool
	// Remote fetch timeout for URL-based configs. Zero disables the timeout.
	FetchTimeout time.Duration
	// Command execution timeout. Zero disables the timeout.
	CommandTimeout time.Duration

	// Operation modes
	Delete   bool
	Reset    bool
	Select   bool
	Yes      bool
	Diff     string
	Replace  bool
	ShowEnv  bool
	Stop     bool
	Launch   bool
	FailFast bool

	// Internal state (not flags)
	FileCount int

	// Incus flags (passed through to incus commands)
	Project    string
	Verbose    bool
	Quiet      bool
	ForceLocal bool
}

// IsDiffOnly returns true when the user only wants to see the diff (no apply).
func (o Options) IsDiffOnly() bool {
	return o.Diff != ""
}

// IsJSONDiff returns true when the diff should be rendered as JSON.
func (o Options) IsJSONDiff() bool {
	return o.Diff == "json"
}
