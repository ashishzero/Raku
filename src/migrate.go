package main

import (
	"Raku/botctx"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
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
	* {
		margin: 0;
		font-family: "Roboto", sans-serif;
		border: 0;
		font-size: 100%;
		font-style: inherit;
		font-weight: inherit;
		margin: 0;
		padding: 0;
		vertical-align: baseline;
		overflow: hidden;
		transition: background-color 0.17s ease, color 0.17s ease;
	}
	
	body {
		background-color: #36393f;
		color: #dcddde;
	}

	a {
		color: #00aff4;
		text-decoration: none
	}

	a:hover {
		text-decoration: underline
	}
	
	.screen {
		width: 75%;
		margin: 0 auto;
	}
	
	.menu-bar {
		background-color: #36393f;
		height: 48px;
		display: flex;
		box-shadow: 0 1px 0 rgba(0, 0, 0, 0.2), 0 2px 0 rgba(0, 0, 0, 0.06);
		padding: 0px 18px;
		color: rgba(255, 255, 255, 0.95);
		line-height: 46px;
		font-size: 18px;
		font-weight: 600;
		z-index: 1;
		width: 100%;
		top: 0;
	}
	
	.menu-bar>.name:before {
		line-height: 0px;
		content: "#";
		margin-right: 6px;
		font-size: 1.1em;
		color: rgba(255, 255, 255, 0.5);
	}
	
	.menu-bar>.topic {
		color: rgba(255, 255, 255, 0.4);
		font-size: 0.75rem;
		font-weight: 400;
		letter-spacing: 0;
		margin-left: 0.4em;
		line-height: 48px;
	}
	
	.menu-bar>.topic:before {
		content: " — ";
		color: rgba(255, 255, 255, 0.2);
	}
	
	.chat-box {
		margin: auto;
		height: calc(100% - 48px);
		overflow-y: auto;
		position: fixed;
		margin-right: 12.5%;
	}
	
	.chat-box::-webkit-scrollbar {
		width: 14px;
		position: absolute;
	}
	
	.chat-box::-webkit-scrollbar-thumb,
	::-webkit-scrollbar-track-piece {
		background-clip: padding-box;
		border: 2.5px solid #36393f;
		border-radius: 7px;
		background-clip: padding-box;
	}
	
	.chat-box::-webkit-scrollbar-thumb {
		background-color: #181a1c;
	}
	
	.chat-box::-webkit-scrollbar-track-piece {
		background-color: rgba(0, 0, 0, 0.25);
	}
	
	.message-group {
		margin: 10px 20px;
		padding: 10px 1px;
	}
	
	.header-group>.avatar img {
		position: absolute;
		border-radius: 50%;
		height: 45px;
		width: 45px;
	}
	
	.header-group>.header {
		margin-left: 60px;
		padding-top: 3px;
		color: #ffffff;
		height: 1.3em;
	}
	
	.header-group>.header>.timestamp {
		color: rgba(255, 255, 255, 0.2);
		font-size: 0.75rem;
		font-weight: 400;
		letter-spacing: 0;
		margin-left: 0.3rem;
	}
	
	.message {
		color: #dcddde;
		margin-left: 60px;
		padding-right: 10px;
	}
	
	.message>.content {
		margin-top: 4px;
		font-size: 0.9375rem;
		color: #dcddde;
	}
	
	.divider {
		border: none;
		border-bottom: 1px solid transparent;
		margin: 20px 0px -20px;
		border-bottom-color: rgba(255, 255, 255, 0.04);
		padding: 15px;
	}
	
	.md>bold {
		font-weight: 700;
	}
	
	.md>underline {
		text-decoration: underline;
	}
	
	.md>strike {
		text-decoration: line-through;
	}
	
	.md>italic {
		font-style: italic;
	}`
)

func mdToHTML(md string) string {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(md))

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	res := markdown.Render(doc, renderer)
	return string(res)
}

func migrateHtmlBuildMessage(builder *strings.Builder, msg *discordgo.Message) {
	builder.WriteString("<div class='header-group'>")
	builder.WriteString("<div class='avatar'>")
	builder.WriteString(fmt.Sprintf("<img src='%+v?size=128' alt='Avatar-%+v'></img>", msg.Author.AvatarURL(""), msg.Author.Username))
	builder.WriteString("</div>")
	builder.WriteString(fmt.Sprintf("<div class='header'>%+v<span class='timestamp'>", msg.Author.Username))
	builder.WriteString(msg.Timestamp.Format("01-02-2006 15:04:05 Monday"))

	builder.WriteString("</span></div>")
	builder.WriteString("</div>")

	content := mdToHTML(msg.Content)

	builder.WriteString("<div class='message'>")

	builder.WriteString("<div class='content'>")
	builder.WriteString(content)
	builder.WriteString("</div>")

	for _, attachment := range msg.Attachments {
		builder.WriteString("<div class='content'>")
		if strings.HasPrefix(attachment.ContentType, "image") {
			img := fmt.Sprintf("<img src='%+v' alt='%+v' class='attached' style='max-width: 256px' >", attachment.URL, attachment.Filename)
			builder.WriteString(img)
		} else {
			link := fmt.Sprintf("<a href='%+v'>%+v</a>", attachment.URL, attachment.Filename)
			builder.WriteString(link)
		}
		builder.WriteString("</div>")
	}

	builder.WriteString("</div>")
	builder.WriteString("<hr class='divider' />")
}

func migrateHtmlBegin(builder *strings.Builder, channel *discordgo.Channel) {
	builder.WriteString("<!DOCTYPE html>")
	builder.WriteString("<html lang='en'>")
	builder.WriteString("<head>")
	builder.WriteString("<meta charset='UTF-8'>")
	builder.WriteString("<meta name='viewport' content='width=device-width, initial-scale=1'>")
	builder.WriteString(fmt.Sprintf("<title>%+v</title>", channel.Name))
	builder.WriteString("<style>")
	builder.WriteString(exportCSS)
	builder.WriteString("</style>")
	builder.WriteString("</head>")

	builder.WriteString("<body>")
	builder.WriteString("<div class='screen'>")
	builder.WriteString("<div class='menu-bar'>")
	builder.WriteString(fmt.Sprintf("<div class='name'>%+v</div>", channel.Name))
	builder.WriteString("</div>")
	builder.WriteString("<div class='chat-box'>")
	builder.WriteString("<div class='message-group'>")
}

func migrateHtmlEnd(builder *strings.Builder) {
	builder.WriteString("</div>")
	builder.WriteString("</div>")
	builder.WriteString("</div>")
	builder.WriteString("</div>")
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
		if !strings.HasSuffix(filename, ".html") && !strings.HasPrefix(filename, ".json") {
			filename += ".json"
		}
	}

	if channel != nil && (channel.Type != discordgo.ChannelTypeGuildText && channel.Type != discordgo.ChannelTypeGuildPublicThread) {
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
	var fileMigrateErr error = nil

	if channel != nil {
		desc = fmt.Sprintf("Filtering complete. Migrating %+v messages to channel: <#%+v>...", len(filtered), channel.ID)
		migrateUpdateResponse(session, i.Interaction, desc, 0x11dddd)

		var webhook *discordgo.Webhook

		parent := channel.ID

		if channel.IsThread() {
			parent = channel.ParentID
		}

		webhook, channelMigrateErr = session.WebhookCreate(parent, "migration-webhook", session.State.User.AvatarURL(""))

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

			if channel.IsThread() {
				_, channelMigrateErr = session.WebhookThreadExecute(webhook.ID, webhook.Token, false, channel.ID, param)
			} else {
				_, channelMigrateErr = session.WebhookExecute(webhook.ID, webhook.Token, false, param)
			}

			if time.Since(start) > 30*time.Second {
				start = time.Now()
				desc = fmt.Sprintf("Migrated %v of %v messages...", idx+1, len(filtered))
				migrateUpdateResponse(session, i.Interaction, desc, 0x11dddd)
			}
		}

		if webhook != nil {
			session.WebhookDelete(webhook.ID)
		}
	}

	var fileBuilder strings.Builder

	if len(filename) > 0 {
		var srcChannel *discordgo.Channel
		srcChannel, fileMigrateErr = session.Channel(i.ChannelID)
		if fileMigrateErr == nil {
			if strings.HasSuffix(filename, ".json") {
				var jsonBytes []byte
				jsonBytes, fileMigrateErr = json.Marshal(filtered)
				if fileMigrateErr == nil {
					fileBuilder.Write(jsonBytes)
				}
			} else {
				migrateHtmlBegin(&fileBuilder, srcChannel)
				for _, msg := range filtered {
					migrateHtmlBuildMessage(&fileBuilder, msg)
				}
				migrateHtmlEnd(&fileBuilder)
			}
		}
	}

	content := ""

	if channelMigrateErr == nil && channel != nil {
		content += fmt.Sprintf("\n:green_circle: Migration from <#%+v> to <#%+v> complete.", i.ChannelID, channel.ID)
	} else if channel != nil {
		log.Printf("error: migrate: %+v\n", channelMigrateErr)
		content += fmt.Sprintf("\n:question: Migration from <#%+v> to <#%+v> failed: %+v", i.ChannelID, channel.ID, channelMigrateErr.Error())
	}

	if fileMigrateErr == nil && len(filename) > 0 {
		content += fmt.Sprintf("\n:green_circle: Migration from <#%+v> to file %+v complete.", i.ChannelID, filename)
	} else if len(filename) > 0 {
		content += fmt.Sprintf("\n:question: Migration from <#%+v> to file %+v failed: %+v", i.ChannelID, filename, fileMigrateErr.Error())
	}

	var files = make([]*discordgo.File, 0, 1)

	if len(filename) > 1 {
		reader := strings.NewReader(fileBuilder.String())
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
