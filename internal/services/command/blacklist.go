package command

type CommandSpec struct {
	Level        string
	Args         int
	AllowedFlags map[string]bool
}

// IsAllowed reports whether the provided command name appears in the
// global whitelist.  It is used by the service to reject requests from
// the language model that try to execute something outside the
// permitted set.
func IsAllowed(cmd string) bool {
	_, ok := allowed[cmd]
	return ok
}

var allowed = map[string]CommandSpec{

	// navigation / session
	"pwd": {
		Level: "read",
		Args:  0,
	},

	"cd": {
		Level: "session",
		Args:  1,
	},

	// file listing
	"ls": {
		Level: "read",
		Args:  1,
		AllowedFlags: map[string]bool{
			"-l": true,
			"-a": true,
			"-h": true,
			"-1": true,
		},
	},

	"stat": {
		Level: "read",
		Args:  1,
	},

	// file read
	"cat": {
		Level: "read",
		Args:  1,
	},

	"head": {
		Level: "read",
		Args:  1,
		AllowedFlags: map[string]bool{
			"-n": true,
		},
	},

	"tail": {
		Level: "read",
		Args:  1,
		AllowedFlags: map[string]bool{
			"-n": true,
		},
	},

	// directory operations
	"mkdir": {
		Level: "write",
		Args:  1,
		AllowedFlags: map[string]bool{
			"-p": true,
		},
	},

	"rmdir": {
		Level: "write",
		Args:  1,
	},

	// file operations
	"touch": {
		Level: "write",
		Args:  1,
	},

	"cp": {
		Level: "write",
		Args:  2,
	},

	"mv": {
		Level: "write",
		Args:  2,
	},

	// file search
	"find": {
		Level: "read",
		Args:  1,
		AllowedFlags: map[string]bool{
			"-name": true,
			"-type": true,
		},
	},

	"grep": {
		Level: "read",
		Args:  2,
		AllowedFlags: map[string]bool{
			"-r": true,
			"-n": true,
			"-i": true,
		},
	},

	// system info
	"whoami": {
		Level: "read",
		Args:  0,
	},

	"id": {
		Level: "read",
		Args:  0,
	},

	"uname": {
		Level: "read",
		Args:  0,
		AllowedFlags: map[string]bool{
			"-a": true,
		},
	},

	"df": {
		Level: "read",
		Args:  0,
		AllowedFlags: map[string]bool{
			"-h": true,
		},
	},

	"du": {
		Level: "read",
		Args:  1,
		AllowedFlags: map[string]bool{
			"-h": true,
			"-s": true,
		},
	},

	"env": {
		Level: "read",
		Args:  0,
	},

	// git tools
	"git_status": {
		Level: "read",
		Args:  0,
	},

	"git_diff": {
		Level: "read",
		Args:  0,
	},

	"git_log": {
		Level: "read",
		Args:  0,
	},

	"git_commit": {
		Level: "write",
		Args:  1,
	},

	// go tools
	"run_go": {
		Level: "execute",
		Args:  1,
	},

	"run_tests": {
		Level: "execute",
		Args:  0,
	},

	// network
	"http_get": {
		Level: "network",
		Args:  1,
	},

	"download_file": {
		Level: "network",
		Args:  2,
	},
}
