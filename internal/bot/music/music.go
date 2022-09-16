package music

import (
	"errors"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// Note(Fredrico):
// We need to make the bot disconnect on inactivity.
// We should also make the guild map internal to better synchronize access

type GuildPlayer struct {
	GuildID         string                     // Required
	VoiceConnection *discordgo.VoiceConnection // Internal
	Volume          int8                       // Optional
}

type ConnectionError string

func (err ConnectionError) Error() string {
	return string("A connection error occured: " + err)
}

func (player *GuildPlayer) Connect(session *discordgo.Session, channelID string) error {
	var err error

	if player.VoiceConnection != nil {
		if player.VoiceConnection.ChannelID == channelID {
			return ConnectionError("The bot is already connected to the given voice channel")
		}

		err = player.VoiceConnection.Disconnect()

		if err != nil {
			return err
		}
	}

	vc, err := session.ChannelVoiceJoin(player.GuildID, channelID, false, false)

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

var allGuildPlayers = map[string]*GuildPlayer{}
var allGuildPlayersMutex = sync.Mutex{}

func GetGuildPlayer(guildID string, createIfNotFound bool) *GuildPlayer {
	allGuildPlayersMutex.Lock()

	player, ok := allGuildPlayers[guildID]

	if !ok && createIfNotFound {
		player = &GuildPlayer{
			GuildID: guildID,
		}
		allGuildPlayers[guildID] = player
	}

	allGuildPlayersMutex.Unlock()

	return player
}

func DeleteGuildPlayer(guildID string) error {
	allGuildPlayersMutex.Lock()

	var err error
	player := allGuildPlayers[guildID]

	if player == nil {
		err = errors.New("Something something")
	} else {
		player.Disconnect()
		delete(allGuildPlayers, guildID)
	}

	allGuildPlayersMutex.Unlock()

	return err
}
