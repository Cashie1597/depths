//go:build linux

package protect

// Minimal Linux hard denylist — service managers, kernel thread parent, remote access, containers.
// Exact basename match only. Intentionally short so DEPTHS does not become annoying.
var osNameDeny = []string{
	"systemd",
	"init",
	"kthreadd",
	"dockerd",
	"containerd",
	"containerd-shim",
	"containerd-shim-runc-v2",
}

var osProtectedPrefixes = []string{
	"/usr/lib/systemd/",
	"/sbin/",
	"/usr/sbin/sshd",
}
