package commands

import (
	"github.com/bwmarrin/discordgo"
)

type Session = discordgo.Session
type Interaction = discordgo.InteractionCreate

type Descriptor = discordgo.ApplicationCommand
type Command struct {
	Info       Descriptor
	Handler    func(s *Session, i *Interaction)
	Registered bool
}

var All = map[string]*Command{}

func (cmd Command) add() {
	All[cmd.Info.Name] = &cmd
}
