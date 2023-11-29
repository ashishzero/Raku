package main

import (
	"Raku/botctx"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var MigrateCommand = botctx.CommandDesc{
	Name:        "migrate",
	Description: "migrate message from this channel to another channel or html file",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionChannel,
			Name:        "channel",
			Description: "name of the channel to migrate",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "filename",
			Description: "name for the migrate html file",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionBoolean,
			Name:        "mention",
			Description: "mention original author in migrated messages",
			Required:    false,
		},
	},
	Func:        migrate,
	Interaction: nil,
}

var (
	exportCSS = `
	body {
		margin: 0 auto;
		max-width: 800px;
		padding: 0 20px;
	  }
	  
	  .container {
		border: 2px solid #dedede;
		background-color: #f1f1f1;
		border-radius: 5px;
		padding: 10px;
		margin: 10px 0;
	  }
	  
	  .darker {
		border-color: #ccc;
		background-color: #ddd;
	  }
	  
	  .container::after {
		content: "";
		clear: both;
		display: table;
	  }
	  
	  .container img.left {
		float: left;
		max-width: 60px;
		width: 100%;
		margin-right: 20px;
		border-radius: 50%;
	  }

	  .container img.attached {
		max-width: 512px;
		width: 100%;
		margin-right: 20px;
	  }
	  
	  .container img.right {
		float: right;
		margin-left: 20px;
		margin-right:0;
	  }
	  
	  .time-right {
		float: right;
		color: #aaa;
	  }
	  
	  .time-left {
		float: left;
		color: #999;
	  }
	`
)

func migrateHtmlBuildMessage(builder *strings.Builder, msg *discordgo.Message) {
	avatar := fmt.Sprintf("<img src='%+v' alt='Avatar' class='left' style='width:100%%;'>", msg.Author.AvatarURL(""))
	content := fmt.Sprintf("<p>%+v</p>", msg.Content)
	timestamp := fmt.Sprintf("<span class='time-right'>%+v</span>", msg.Timestamp.String())

	builder.WriteString("<div class='container darker'>")
	builder.WriteString(avatar)
	builder.WriteString(content)
	builder.WriteString(timestamp)

	for _, attachment := range msg.Attachments {
		if strings.HasPrefix(attachment.ContentType, "image") {
			img := fmt.Sprintf("<img src='%+v' alt='%+v' class='attached'>", attachment.URL, attachment.Filename)
			builder.WriteString(img)
		} else {
			link := fmt.Sprintf("<a href='%+v'>%+v</a>", attachment.URL, attachment.Filename)
			builder.WriteString(link)
		}
	}

	builder.WriteString("</div>")
}

func migrateHtmlBegin(builder *strings.Builder, header string) {
	builder.WriteString("<!DOCTYPE html>")
	builder.WriteString("<html>")
	builder.WriteString("<head>")
	builder.WriteString("<meta name='viewport' content='width=device-width, initial-scale=1'>")
	builder.WriteString("<style>")
	builder.WriteString(exportCSS)
	builder.WriteString("</style>")
	builder.WriteString("</head>")
	builder.WriteString("<body>")
	builder.WriteString("<h2>")
	builder.WriteString(header)
	builder.WriteString("</h2>")
	builder.WriteString("\n")
}

func migrateHtmlEnd(builder *strings.Builder) {
	builder.WriteString("\n")
	builder.WriteString("</body>")
	builder.WriteString("</html>")
}

func migrateUpdateResponse(session *discordgo.Session, interaction *discordgo.Interaction, desc string, color int) bool {
	edit := &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       "Migration",
				Description: desc,
				Type:        discordgo.EmbedTypeRich,
				Color:       color,
			},
		},
	}
	_, err := session.InteractionResponseEdit(interaction, edit)
	if err != nil {
		log.Printf("error: migrateUpdateResponse: %+v\n", err)
		return false
	}
	return true
}

func migrate(session *discordgo.Session, i *discordgo.InteractionCreate) {
	var filename string = ""
	var channel *discordgo.Channel
	mention := true

	data := i.ApplicationCommandData()
	for _, option := range data.Options {
		if option.Name == "filename" {
			filename = option.StringValue()
		} else if option.Name == "channel" {
			channel = option.ChannelValue(session)
		} else if option.Name == "mention" {
			mention = option.BoolValue()
		}
	}

	if filename == "" && channel == nil {
		var res = &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Migration Failed",
						Description: "One of channel name or file name is required",
						Type:        discordgo.EmbedTypeRich,
						Color:       0xdd1111,
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		}
		session.InteractionRespond(i.Interaction, res)
		return
	}

	if len(filename) > 0 {
		if !strings.HasSuffix(filename, ".html") {
			filename += ".html"
		}
	}

	if channel != nil && channel.Type != discordgo.ChannelTypeGuildText {
		var res = &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Migration Failed",
						Description: fmt.Sprintf("Invalid channel %+v only text channel are accepted", channel.Name),
						Color:       0xdd1111,
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		}
		session.InteractionRespond(i.Interaction, res)
		return
	}

	{
		res := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Migration",
						Description: "Downloading messages...",
						Type:        discordgo.EmbedTypeRich,
						Color:       0x11dddd,
					},
				},
			},
		}
		err := session.InteractionRespond(i.Interaction, res)
		if err != nil {
			log.Printf("error: migrate: %+v\n", err)
			return
		}
	}

	beforeID := ""

	var msgs = make([]*discordgo.Message, 0, 1024*1024)

	var start = time.Now()

	for {
		downs, err := session.ChannelMessages(i.ChannelID, 100, beforeID, "", "")
		if err != nil {
			if !migrateUpdateResponse(session, i.Interaction, "Failed to download message beforeID: "+beforeID, 0xff2222) {
				return
			}
		}
		if len(downs) == 0 {
			break
		}
		beforeID = downs[len(downs)-1].ID
		msgs = append(msgs, downs...)

		if time.Since(start) > 30*time.Second {
			start = time.Now()
			desc := fmt.Sprintf("Downloading messages (%v)...", len(msgs))
			if !migrateUpdateResponse(session, i.Interaction, desc, 0x11dddd) {
				return
			}
		}
	}

	desc := fmt.Sprintf("Downloaded (%v) messages. Filtering messages...", len(msgs))
	if !migrateUpdateResponse(session, i.Interaction, desc, 0x11dddd) {
		return
	}

	filtered := make([]*discordgo.Message, 0, len(msgs))

	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Author.Bot {
			continue
		}
		if len(msgs[i].Content) == 0 && len(msgs[i].Attachments) == 0 {
			continue
		}
		filtered = append(filtered, msgs[i])
	}

	var channelMigrateErr error = nil
	var htmlMigrateErr error = nil

	if channel != nil {
		desc = fmt.Sprintf("Filtering complete. Migrating %+v messages to channel: <#%+v>...", len(filtered), channel.ID)
		migrateUpdateResponse(session, i.Interaction, desc, 0x11dddd)

		var webhook *discordgo.Webhook
		webhook, channelMigrateErr = session.WebhookCreate(channel.ID, "migration-webhook", session.State.User.AvatarURL(""))

		for idx, msg := range filtered {
			if channelMigrateErr != nil {
				break
			}

			content := msg.Content
			if mention {
				content += "\n\n- *Original Author*: <@" + msg.Author.ID + ">\n\n"
			}

			attachments := make([]*discordgo.File, 0, len(msg.Attachments))

			for _, src := range msg.Attachments {
				res, err := session.Client.Get(src.URL)
				if err != nil {
					continue
				}
				attachments = append(attachments, &discordgo.File{
					Name:        src.Filename,
					ContentType: src.ContentType,
					Reader:      res.Body,
				})
				defer res.Body.Close()
			}

			param := &discordgo.WebhookParams{
				Content:   content,
				Username:  msg.Author.Username,
				AvatarURL: msg.Author.AvatarURL(""),
				Embeds:    msg.Embeds,
				Files:     attachments,
				AllowedMentions: &discordgo.MessageAllowedMentions{
					Parse: []discordgo.AllowedMentionType{},
				},
			}

			_, channelMigrateErr = session.WebhookExecute(webhook.ID, webhook.Token, false, param)

			if time.Since(start) > 30*time.Second {
				start = time.Now()
				desc = fmt.Sprintf("Migrated %v of %v messages...", idx+1, len(filtered))
				migrateUpdateResponse(session, i.Interaction, desc, 0x11dddd)
			}
		}

		session.WebhookDelete(webhook.ID)
	}

	var htmlBuilder strings.Builder

	if len(filename) > 0 {
		srcChannel, htmlMigrateErr := session.Channel(i.ChannelID)
		if htmlMigrateErr == nil {
			migrateHtmlBegin(&htmlBuilder, srcChannel.Name)
			for _, msg := range filtered {
				migrateHtmlBuildMessage(&htmlBuilder, msg)
			}
			migrateHtmlEnd(&htmlBuilder)
		}
	}

	content := ""

	if channelMigrateErr == nil && channel != nil {
		content += fmt.Sprintf("\n:green_circle: Migration from <#%+v> to <#%+v> complete.", i.ChannelID, channel.ID)
	} else if channel != nil {
		log.Printf("error: migrate: %+v\n", channelMigrateErr)
		content += fmt.Sprintf("\n:negative_squared_cross_mark: Migration from <#%+v> to <#%+v> failed.", i.ChannelID, channel.ID)
	}

	if htmlMigrateErr == nil && len(filename) > 0 {
		content += fmt.Sprintf("\n:green_circle: Migration from <#%+v> to file %+v complete.", i.ChannelID, filename)
	} else if len(filename) > 0 {
		content += fmt.Sprintf("\n:negative_squared_cross_mark: Migration from <#%+v> to file %+v failed.", i.ChannelID, filename)
	}

	var files = make([]*discordgo.File, 0, 1)

	if len(filename) > 1 {
		reader := strings.NewReader(htmlBuilder.String())
		files = append(files, &discordgo.File{
			Name:   filename,
			Reader: reader,
		})
	}

	res := &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       "Migration",
				Description: content,
				Type:        discordgo.EmbedTypeRich,
				Color:       0x11ff22,
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Total %+v messages", len(filtered)),
				},
			},
		},
		Files: files,
	}

	_, err := session.InteractionResponseEdit(i.Interaction, res)
	if err != nil {
		log.Printf("error: migrate: %+v\n", err)
	}
}
