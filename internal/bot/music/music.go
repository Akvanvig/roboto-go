package music

import "github.com/bwmarrin/discordgo"

// Note(Fredrico):
// We need to make the bot disconnect on inactivity.
// We should also make the guild map internal to better synchronize access

type GuildPlayer struct {
	GuildID         string                     // Required
	ChannelID       string                     // Required
	Volume          int8                       // Optional
	VoiceConnection *discordgo.VoiceConnection // Internal
}

func (player *GuildPlayer) Connect(session *discordgo.Session) error {
	vc, err := session.ChannelVoiceJoin(player.GuildID, player.ChannelID, false, false)

	if err != nil {
		return err
	}

	player.VoiceConnection = vc
	return nil
}

func (player *GuildPlayer) Disconnect() error {
	vc := player.VoiceConnection

	if vc == nil {
		return nil
	}

	err := vc.Disconnect()

	if err != nil {
		return err
	}

	return nil
}

func (player *GuildPlayer) Play(videoName string) {

}

var GuildPlayers = map[string]*GuildPlayer{}
