package command

// blocked contains commands that are explicitly disallowed.  The
// service will execute anything *not* in this list; a given command is
// rejected only when it appears here.
var blocked = map[string]struct{}{
	// destructive filesystem
	"rm":        {},
	"shred":     {},
	"wipe":      {},
	"srm":       {},
	"dd":        {},
	"truncate":  {},
	"mkfs":      {},
	"mkfs.ext4": {},
	"mkfs.xfs":  {},
	"mke2fs":    {},

	// powerful disk / partition tools
	"fdisk":      {},
	"gdisk":      {},
	"parted":     {},
	"lsblk":      {},
	"blkdiscard": {},
	"blkid":      {},
	"losetup":    {},
	"fdformat":   {},

	// mount / unmount / loop / device nodes
	"mount":     {},
	"umount":    {},
	"mount.nfs": {},
	"mknod":     {},

	// system / init / power / kernel modules
	"reboot":    {},
	"shutdown":  {},
	"halt":      {},
	"poweroff":  {},
	"systemctl": {},
	"service":   {},
	"init":      {},
	"telinit":   {},
	"modprobe":  {},
	"insmod":    {},
	"rmmod":     {},
	"sysctl":    {},

	// package managers (Arch, Debian, RedHat, etc.)
	"pacman":  {},
	"apt":     {},
	"apt-get": {},
	"dnf":     {},
	"yum":     {},
	"zypper":  {},
	"rpm":     {},
	"emerge":  {},

	// privileged user management
	"sudo":     {},
	"su":       {},
	"useradd":  {},
	"userdel":  {},
	"usermod":  {},
	"groupadd": {},
	"groupdel": {},
	"passwd":   {},

	// networking / firewall / tunneling / scanners
	"iptables":  {},
	"ip6tables": {},
	"nft":       {},
	"ip":        {},
	"ifconfig":  {},
	"route":     {},
	"tc":        {},
	"iw":        {},
	"iwconfig":  {},
	"nmcli":     {},
	"nmap":      {},
	"nc":        {}, // netcat
	"netcat":    {},
	"socat":     {},
	"ssh":       {}, // raw ssh executables — prefer controlled wrappers if needed
	"scp":       {},
	"rsync":     {},

	// raw download/execution utilities — prefer controlled http_get/download wrapper
	"curl":  {},
	"wget":  {},
	"fetch": {},

	// container / virtualization / orchestration
	"docker":   {},
	"podman":   {},
	"runc":     {},
	"kubectl":  {},
	"crictl":   {},
	"lxc":      {},
	"virsh":    {},
	"qemu-img": {},

	// interpreters and shells (prevent arbitrary script execution)
	"bash":    {},
	"sh":      {},
	"zsh":     {},
	"dash":    {},
	"python":  {},
	"python3": {},
	"perl":    {},
	"ruby":    {},
	"node":    {},
	"php":     {},

	// build / system tools that can be misused
	"make":  {},
	"gcc":   {},
	"g++":   {},
	"strip": {},

	// crypto / key tools (can expose/private-key ops)
	"openssl": {},
	"gpg":     {},

	// database servers / clients (could expose/modify data)
	"mysql":     {},
	"mysqld":    {},
	"psql":      {},
	"mongo":     {},
	"redis-cli": {},

	// low-level system utilities
	"chattr":  {},
	"fuser":   {},
	"kill":    {},
	"killall": {},

	// other potentially dangerous utilities
	"echo": {},
	">":    {},
}

// IsBlocked reports whether the provided command name appears in the
// blacklist.
func IsBlocked(cmd string) bool {
	_, ok := blocked[cmd]
	return ok
}

// IsAllowed returns true for commands that are not blocked.  It exists
// mainly for backwards‑compatible callers that expect the old semantics.
func IsAllowed(cmd string) bool {
	return !IsBlocked(cmd)
}
