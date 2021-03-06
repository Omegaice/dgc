package dgc

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// RegisterDefaultHelpCommand registers the default help command
func (router *Router) RegisterDefaultHelpCommand(session *discordgo.Session, rateLimiter RateLimiter) {
	// Initialize the helo messages storage
	router.InitializeStorage("dgc_helpMessages")

	// Initialize the reaction add listener
	session.AddHandler(func(session *discordgo.Session, event *discordgo.MessageReactionAdd) {
		// Define useful variables
		channelID := event.ChannelID
		messageID := event.MessageID
		userID := event.UserID

		// Check whether or not the reaction was added by the bot itself
		if event.UserID == session.State.User.ID {
			return
		}

		// Check whether or not the message is a help message
		rawPage, ok := router.Storage["dgc_helpMessages"].Get(channelID + ":" + messageID + ":" + event.UserID)
		if !ok {
			return
		}
		page := rawPage.(int)
		if page <= 0 {
			return
		}

		// Check which reaction was added
		reactionName := event.Emoji.Name
		switch reactionName {
		case "⬅️":
			// Update the help message
			embed, newPage := renderDefaultGeneralHelpEmbed(router, page-1)
			page = newPage
			session.ChannelMessageEditEmbed(channelID, messageID, embed)

			// Remove the reaction
			session.MessageReactionRemove(channelID, messageID, reactionName, userID)
			break
		case "❌":
			// Delete the help message
			session.ChannelMessageDelete(channelID, messageID)
			break
		case "➡️":
			// Update the help message
			embed, newPage := renderDefaultGeneralHelpEmbed(router, page+1)
			page = newPage
			session.ChannelMessageEditEmbed(channelID, messageID, embed)

			// Remove the reaction
			session.MessageReactionRemove(channelID, messageID, reactionName, userID)
			break
		}

		// Update the stores page
		router.Storage["dgc_helpMessages"].Set(channelID+":"+messageID+":"+event.UserID, page)
	})

	// Register the default help command
	router.RegisterCmd(&Command{
		Name:        "help",
		Description: "Lists all the available commands or displays some information about a specific command",
		Usage:       "help [command name]",
		Example:     "help yourCommand",
		IgnoreCase:  true,
		Hidden:      false,
		RateLimiter: rateLimiter,
		Handler:     generalHelpCommand,
	})
}

// generalHelpCommand handles the general help command
func generalHelpCommand(ctx *Ctx) {
	// Check if the user provided an argument
	if ctx.Arguments.Amount() > 0 {
		specificHelpCommand(ctx)
		return
	}

	// Define useful variables
	channelID := ctx.Event.ChannelID
	session := ctx.Session

	// Send the general help embed
	embed, _ := renderDefaultGeneralHelpEmbed(ctx.Router, 1)
	message, _ := ctx.Session.ChannelMessageSendEmbed(channelID, embed)

	// Add the reactions to the message
	session.MessageReactionAdd(channelID, message.ID, "⬅️")
	session.MessageReactionAdd(channelID, message.ID, "❌")
	session.MessageReactionAdd(channelID, message.ID, "➡️")

	// Define the message as a help message
	ctx.Router.Storage["dgc_helpMessages"].Set(channelID+":"+message.ID+":"+ctx.Event.Author.ID, 1)
}

// specificHelpCommand handles the specific help command
func specificHelpCommand(ctx *Ctx) {
	// Define the command names
	commandNames := strings.Split(ctx.Arguments.Raw(), " ")

	// Define the command
	var command *Command
	for index, commandName := range commandNames {
		if index == 0 {
			c := ctx.Router.GetCmd(commandName)
			if !c.Hidden {
				command = c
			}
			continue
		}
		c := command.GetSubCmd(commandName)
		if !c.Hidden {
			command = c
		}
	}

	// Send the help embed
	ctx.Session.ChannelMessageSendEmbed(ctx.Event.ChannelID, renderDefaultSpecificHelpEmbed(ctx, command))
}

// renderDefaultGeneralHelpEmbed renders the general help embed on the given page
func renderDefaultGeneralHelpEmbed(router *Router, page int) (*discordgo.MessageEmbed, int) {
	// Define useful variables
	commands := router.Commands
	prefix := router.Prefixes[0]

	vCommands := []*Command{}

	for _, command := range commands {
		if !command.Hidden {
			vCommands = append(vCommands, command)
		}
	}

	// Calculate the amount of pages
	pageAmount := int(math.Ceil(float64(len(vCommands)) / 5))
	if page > pageAmount {
		page = pageAmount
	}
	if page <= 0 {
		page = 1
	}

	// Calculate the slice of commands to display on this page
	startingIndex := (page - 1) * 5
	endingIndex := startingIndex + 5
	if page == pageAmount {
		endingIndex = len(vCommands)
	}
	displayCommands := vCommands[startingIndex:endingIndex]

	// Prepare the fields for the embed
	fields := []*discordgo.MessageEmbedField{}
	for _, command := range displayCommands {
		if !command.Hidden {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   command.Name,
				Value:  "`" + command.Description + "`",
				Inline: false,
			})
		}
	}

	// Return the embed and the new page
	return &discordgo.MessageEmbed{
		Type:        "rich",
		Title:       "Command List (Page " + strconv.Itoa(page) + "/" + strconv.Itoa(pageAmount) + ")",
		Description: "These are all the available commands. Type `" + prefix + "help <command name>` to find out more about a specific command.",
		Timestamp:   time.Now().Format(time.RFC3339),
		Color:       0xffff00,
		Fields:      fields,
	}, page
}

// renderDefaultSpecificHelpEmbed renders the specific help embed of the given command
func renderDefaultSpecificHelpEmbed(ctx *Ctx, command *Command) *discordgo.MessageEmbed {
	// Define useful variables
	prefix := ctx.Router.Prefixes[0]

	// Check if the command is invalid
	if command == nil {
		return &discordgo.MessageEmbed{
			Type:      "rich",
			Title:     "Error",
			Timestamp: time.Now().Format(time.RFC3339),
			Color:     0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Message",
					Value:  "```The given command doesn't exist. Type `" + prefix + "help` for a list of available commands.```",
					Inline: false,
				},
			},
		}
	}

	toSend := &discordgo.MessageEmbed{
		Type:        "rich",
		Title:       "Command Information",
		Description: "Displaying the information for the `" + command.Name + "` command.",
		Timestamp:   time.Now().Format(time.RFC3339),
		Color:       0xffff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Name",
				Value:  "`" + command.Name + "`",
				Inline: false,
			},
		},
	}

	// Define the sub commands string
	if len(command.SubCommands) > 0 {
		subCommandNames := []string{}
		for _, subCommand := range command.SubCommands {
			if !subCommand.Hidden {
				subCommandNames = append(subCommandNames, subCommand.Name)
			}
		}

		toSend.Fields = append(toSend.Fields, &discordgo.MessageEmbedField{
			Name:   "Sub Commands",
			Value:  "`" + strings.Join(subCommandNames, "`, `") + "`",
			Inline: false,
		})
	}

	// Define the aliases string
	if len(command.Aliases) > 0 {
		toSend.Fields = append(toSend.Fields, &discordgo.MessageEmbedField{
			Name:   "Aliases",
			Value:  "`" + strings.Join(command.Aliases, "`, `") + "`",
			Inline: false,
		})
	}

	if len(command.Description) > 0 {
		toSend.Fields = append(toSend.Fields, &discordgo.MessageEmbedField{
			Name:   "Description",
			Value:  "```" + command.Description + "```",
			Inline: false,
		})
	}

	if len(command.Usage) > 0 {
		toSend.Fields = append(toSend.Fields, &discordgo.MessageEmbedField{
			Name:   "Usage",
			Value:  "```" + prefix + command.Usage + "```",
			Inline: false,
		})
	}

	if len(command.Example) > 0 {
		toSend.Fields = append(toSend.Fields, &discordgo.MessageEmbedField{
			Name:   "Example",
			Value:  "```" + prefix + command.Example + "```",
			Inline: false,
		})
	}

	return toSend
}
