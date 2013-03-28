// replies when someone mentions the bot's name
package identity

import (
	"gesture/plugin"
	"strings"
)

type Plugin struct {
	name string
}

func NewPlugin(name string) Plugin {
	return Plugin{name: name}
}

func (me Plugin) Call(mc plugin.MessageContext) (bool, error) {
	for _, token := range strings.Split(mc.Message(), " ") {
		if token == me.name {
			mc.Reply("i am halping")
		}
	}
	return false, nil
}