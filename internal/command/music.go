package command

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/Akvanvig/roboto-go/internal/player"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
	"github.com/rs/zerolog/log"
)

// See https://github.com/lavalink-devs/youtube-source/blob/ae2b8b316bcd2b2188652d682d2f7fb7dcbbcfd3/common/src/main/java/dev/lavalink/youtube/YoutubeAudioSourceManager.java#L42
var RegexpYoutubeURL *regexp.Regexp
var RegexpYoutubeURLAlt *regexp.Regexp

func init() {
	protocol := "(?:http://|https://|)"
	domain := "(?:www\\.|m\\.|music\\.|)youtube\\.com"
	domainShort := "(?:www\\.|)youtu\\.be"

	regexpYoutube, err := regexp.Compile("^" + protocol + domain + "/.*")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to compile youtube regexp")
	}

	regexpYoutubeAlt, err := regexp.Compile("^" + protocol + "(?:" + domain + "/(?:live|embed|shorts)|" + domainShort + ")/(?<videoId>.*)")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to compile youtube regexp")
	}

	RegexpYoutubeURL = regexpYoutube
	RegexpYoutubeURLAlt = regexpYoutubeAlt
}

// -- BOOTSTRAP --

func musicCommands(bot *bot.RobotoBot, r *handler.Mux) discord.ApplicationCommandCreate {
	if bot.Player == nil {
		return nil
	}

	cmds := discord.SlashCommandCreate{
		Name:        "music",
		Description: "Music commands",
		Contexts: []discord.InteractionContextType{
			discord.InteractionContextTypeGuild,
		},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "play",
				Description: "Play some music",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "query",
						Description: "The search query",
						Required:    true,
					},
					discord.ApplicationCommandOptionString{
						Name:        "source",
						Description: "The alternative search source to use, default is YouTube",
						Choices: []discord.ApplicationCommandOptionChoiceString{
							{
								Name:  "YouTube Music",
								Value: string(lavalink.SearchTypeYouTubeMusic),
							},
							{
								Name:  "SoundCloud",
								Value: string(lavalink.SearchTypeSoundCloud),
							},
						},
					},
				},
			},
			/*
				discord.ApplicationCommandOptionSubCommand{
					Name:        "filter",
					Description: "Toggle music filters",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "filter",
							Description: "The filter to toggle",
							Required:    true,
							Choices: []discord.ApplicationCommandOptionChoiceString{
								{
									Name:  "Karaoke",
									Value: string(player.FilterTypeKaraoke),
								},
								{
									Name:  "Vibrato",
									Value: string(player.FilterTypeVibrato),
								},
							},
						},
					},
				},*/
			discord.ApplicationCommandOptionSubCommand{
				Name:        "volume",
				Description: "Adjust the music volume",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionInt{
						Name:        "number",
						Description: "The volume percentage",
						Required:    true,
						MinValue:    json.Ptr(0),
						MaxValue:    json.Ptr(100),
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "clear",
				Description: "Clear the music queue",
			},
		},
	}

	h := &MusicHandler{
		Player: bot.Player,
	}
	r.Route("/music", func(r handler.Router) {
		r.SlashCommand("/play", h.onPlay)
		r.Group(func(r handler.Router) {
			r.Use(func(next handler.Handler) handler.Handler {
				return func(e *handler.InteractionEvent) error {
					client := e.Client()

					channelID := h.Player.ChannelID(*e.GuildID())
					if channelID == nil {
						return e.Respond(discord.InteractionResponseTypeCreateMessage, discord.MessageUpdate{
							Embeds: json.Ptr(Embeds("No music is currently playing", MessageColorError)),
							Flags:  json.Ptr(discord.MessageFlagEphemeral),
						})
					}

					caches := client.Caches()
					if *channelID != e.Channel().ID() {
						channel, _ := caches.Channel(*channelID)
						return e.Respond(discord.InteractionResponseTypeCreateMessage, discord.MessageUpdate{
							Embeds: json.Ptr(Embeds(fmt.Sprintf("Expecting music interactions in the %s channel", channel.Mention()), MessageColorError)),
							Flags:  json.Ptr(discord.MessageFlagEphemeral),
						})
					}

					vsUser, userOk := caches.VoiceState(*e.GuildID(), e.User().ID)
					vsBot, botOk := caches.VoiceState(*e.GuildID(), e.ApplicationID())
					if botOk {
						if !userOk || (*vsUser.ChannelID != *vsBot.ChannelID) {
							return e.Respond(discord.InteractionResponseTypeCreateMessage, discord.MessageUpdate{
								Embeds: json.Ptr(Embeds("Must be in the same voice channel as the bot to interact with it", MessageColorError)),
								Flags:  json.Ptr(discord.MessageFlagEphemeral),
							})
						}
					}

					return next(e)
				}
			})

			//r.SlashCommand("/filter", h.onFilter)
			r.SlashCommand("/volume", h.onVolume)
			r.SlashCommand("/clear", h.onClear)
			r.Component("/skip", h.onSkipButton)
			r.Component("/stop", h.onStopButton)
			r.Component("/queue", h.onQueueButton)
		})
	})

	return cmds
}

// -- HANDLERS --

type MusicHandler struct {
	Player *player.Player
}

func (h *MusicHandler) onPlay(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	client := e.Client()
	vsUser, ok := client.Caches().VoiceState(*e.GuildID(), e.User().ID)
	if !ok {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Must be in a voice channel to queue songs", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	src := data.String("source")
	q := data.String("query")

	switch src {
	case string(lavalink.SearchTypeSoundCloud):
		q = lavalink.SearchTypeSoundCloud.Apply(q)
	case string(lavalink.SearchTypeYouTubeMusic):
		q = lavalink.SearchTypeYouTubeMusic.Apply(q)
	default:
		// If the query is a direct link, we just send the url directly
		if !RegexpYoutubeURL.MatchString(q) && !RegexpYoutubeURLAlt.MatchString(q) {
			q = lavalink.SearchTypeYouTube.Apply(q)
		}
	}

	err := e.DeferCreateMessage(false)
	if err != nil {
		return err
	}

	err = h.Player.Search(e.Ctx, *e.GuildID(), q,
		func(tracks ...lavalink.Track) {
			if len(tracks) == 0 {
				e.UpdateInteractionResponse(discord.MessageUpdate{
					Embeds: json.Ptr(Embeds(fmt.Sprintf("No results found for %s", q), MessageColorDefault)),
				})
				return
			}

			vsBot, ok := client.Caches().VoiceState(*e.GuildID(), e.ApplicationID())
			if ok {
				if *vsUser.ChannelID != *vsBot.ChannelID {
					e.UpdateInteractionResponse(discord.MessageUpdate{
						Embeds: json.Ptr(Embeds("Must be in the same voice channel as the bot to interact with it", MessageColorError)),
					})
					return
				}
			} else {
				err := client.UpdateVoiceState(context.Background(), *e.GuildID(), vsUser.ChannelID, false, false)
				if err != nil {
					e.UpdateInteractionResponse(discord.MessageUpdate{
						Embeds: json.Ptr(Embeds(err.Error(), MessageColorError)),
					})
					return
				}
			}

			err := h.Player.Add(e.Ctx, *e.GuildID(), e.Channel().ID(), e.User(), tracks...)
			if err != nil {
				e.UpdateInteractionResponse(discord.MessageUpdate{
					Embeds: json.Ptr(Embeds(err.Error(), MessageColorError)),
				})
				return
			}

			e.UpdateInteractionResponse(discord.MessageUpdate{
				Embeds: json.Ptr(player.Embeds("Added to queue", true, tracks...)),
			})
		},
		func(err error) {
			e.UpdateInteractionResponse(discord.MessageUpdate{
				Embeds: json.Ptr(Embeds(strings.TrimPrefix(err.Error(), "fault: "), MessageColorError)),
			})
		},
	)
	if err != nil {
		_, err = e.UpdateInteractionResponse(discord.MessageUpdate{
			Embeds: json.Ptr(Embeds("Failed to search for song", MessageColorError)),
		})
		return err
	}

	return nil
}

func (h *MusicHandler) onFilter(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	filter := player.FilterType(data.String("filter"))
	enabled, err := h.Player.Filter(e.Ctx, *e.GuildID(), filter)
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to toggle filter", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	var text string
	if enabled {
		text = "Enabled"
	} else {
		text = "Disabled"
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: Embeds(fmt.Sprintf("%s %s filter", text, filter), MessageColorDefault),
	})
}

func (h *MusicHandler) onVolume(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	volume := data.Int("number")
	err := h.Player.Volume(e.Ctx, *e.GuildID(), volume)
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to adjust volume", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: Embeds(fmt.Sprintf("Set volume to %d%%", volume), MessageColorDefault),
	})
}

func (h *MusicHandler) onSkipButton(e *handler.ComponentEvent) error {
	track, err := h.Player.Skip(e.Ctx, *e.GuildID(), 1)
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to skip current song", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}
	if track == nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Queue is currently empty", MessageColorDefault),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	e.Acknowledge()
	return nil
}

func (h *MusicHandler) onClear(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	err := h.Player.Clear(e.Ctx, *e.GuildID())
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to clear queue", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: Embeds(fmt.Sprintf("%s cleared the queue", e.User().Mention()), MessageColorDefault),
	})
}

func (h *MusicHandler) onStopButton(e *handler.ComponentEvent) error {
	client := e.Client()

	err := client.UpdateVoiceState(e.Ctx, *e.GuildID(), nil, false, false)
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to disconnect bot from the voice channel", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: Embeds(fmt.Sprintf("%s stopped the music", e.User().Mention()), MessageColorDefault),
	})
}

func (h *MusicHandler) onQueueButton(e *handler.ComponentEvent) error {
	tracks, err := h.Player.Queue(e.Ctx, *e.GuildID())
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to get the current queue", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	if len(tracks) == 0 {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Queue is currently empty", MessageColorDefault),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: player.Embeds("Next up", true, tracks...),
		Flags:  discord.MessageFlagEphemeral,
	})
}
