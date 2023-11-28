package botctx

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type CommandFunc func(session *discordgo.Session, i *discordgo.InteractionCreate)
type InteractionFunc func(session *discordgo.Session, i *discordgo.InteractionCreate, args []string)

type Command struct {
	Command     *discordgo.ApplicationCommand
	Func        CommandFunc
	Interaction InteractionFunc
}

type CommandDesc struct {
	Name        string
	Description string
	Func        CommandFunc
	Interaction InteractionFunc
	Options     []*discordgo.ApplicationCommandOption
}

var (
	commandLUT = make(map[string]Command)
	commands   = make([]*discordgo.ApplicationCommand, 0, 16)
)

func Login(token string) {
	bot, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("error: %+v", err)
	}

	for _, v := range commandLUT {
		commands = append(commands, v.Command)
	}

	bot.AddHandler(ready)
	bot.AddHandler(interactionCreate)

	bot.Identify.Intents = discordgo.IntentGuildMessages | discordgo.IntentGuildIntegrations

	if err != nil {
		log.Fatalln("Failed to create bot", err)
	}

	err = bot.Open()
	if err != nil {
		log.Fatalln("Failed to open bot", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	bot.Close()
}

func RegisterApplicationCommand(desc CommandDesc) {
	commandLUT[desc.Name] = Command{
		Command: &discordgo.ApplicationCommand{
			Name:        desc.Name,
			Description: desc.Description,
			Options:     desc.Options,
		},
		Func:        desc.Func,
		Interaction: desc.Interaction,
	}
}

//
//
//

func ready(session *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as %v#%v", r.User.Username, r.User.Discriminator)
	for _, guild := range r.Guilds {
		session.ApplicationCommandBulkOverwrite(session.State.User.ID, guild.ID, commands)
	}
}

func guildCreate(session *discordgo.Session, e *discordgo.GuildCreate) {
	session.ApplicationCommandBulkOverwrite(session.State.User.ID, e.Guild.ID, commands)
}

func interactionCreate(session *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		data := i.ApplicationCommandData()
		cmd, ok := commandLUT[data.Name]
		if ok {
			cmd.Func(session, i)
		}
	} else if i.Type == discordgo.InteractionMessageComponent {
		data := i.MessageComponentData()
		args := strings.Split(data.CustomID, ";")
		cmd, ok := commandLUT[args[0]]
		if ok {
			cmd.Interaction(session, i, args[1:])
		}
	}
}
