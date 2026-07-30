//go:build darwin

package protect

// Minimal Darwin hard denylist — system-critical + session managers.
// Exact basename match only (see Guard.Check).
var osNameDeny = []string{
	"kernel_task",
	"launchd",
	"WindowServer",
	"loginwindow",
	"UserEventAgent",
	"SystemUIServer",
	"Dock",
	"Finder",
	"cfprefsd",
	"opendirectoryd",
	"securityd",
	"trustd",
	"amfid",
	"syspolicyd",
	"coreservicesd",
	"configd",
	"powerd",
	"logd",
	"notifyd",
	"tccd",
	"sandboxd",
	"secd",
	"backupd",
	"mds",
	"mds_stores",
	"mdworker",
	"mdworker_shared",
}

var osProtectedPrefixes = []string{
	"/System/",
	"/usr/libexec/",
	"/sbin/",
}
