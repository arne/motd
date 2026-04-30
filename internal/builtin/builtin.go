package builtin

import (
	"github.com/arne/motd/internal/module"
)

func Register() {
	module.RegisterBuiltin("hostname", hostname)
	module.RegisterBuiltin("uptime", uptime)
	module.RegisterBuiltin("load", load)
	module.RegisterBuiltin("mem", mem)
	module.RegisterBuiltin("ip", ipAddrs)
	module.RegisterBuiltin("disk", disk)
	module.RegisterBuiltinFull("disk", diskFull)
	module.RegisterBuiltin("services", services)
	module.RegisterBuiltinFull("services", servicesFull)
}
