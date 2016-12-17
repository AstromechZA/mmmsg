package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/textproto"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	// mattermost bot client
	"github.com/mattermost/platform/model"

	"mime"

	"github.com/AstromechZA/mmmsg/conf"
)

const usageString = `mmmsg is a slim binary for pushing a mattermost message to
a particular channel or user. Mattermost url and credentials are provided from
a config file while the message content is read from stdin.

Config File Json:
{
    "mattermost_api": "<mattermost url WITHOUT /api/json>",
    "mattermost_user": "<username or email address>",
    "mattermost_password": "<password>",
    "mattermost_team": "<team name>",
    "default_channel": "<default channel if not provided>"
}

`

const logoImage = `
    ____________________________________________________
   T ================================================= |T
   | ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~||
   | __________________________________________________[|
   |I __==___________  ___________                 __  T|
   ||[_j  L_I_I_I_I_j  L_I_I_I_I_j                 ==  l|
   lI _______________________________  _____  _________I]
    |[__I_I_I_I_I_I_I_I_I_I_I_I_I_I_] [__I__] [_I_I_I_]|
    |[___I_I_I_I_I_I_I_I_I_I_I_I_L  I   ___   [_I_I_I_]|
    |[__I_I_I_I_I_I_I_I_I_I_I_I_I_L_I __I_]_  [_I_I_T ||
    |[___I_I_I_I_I_I_I_I_I_I_I_I____] [_I_I_] [___I_I_j|
    | [__I__I_________________I__L_]                   |
    |                                                  |
    l__________________________________________________j
`

// MMMsgVersion is the version string
// format should be 'X.YZ'
// Set this at build time using the -ldflags="-X main.MMMsgVersion=X.YZ"
var MMMsgVersion = "<unofficial build>"

func mainInner() error {

	// first set up config flag options
	configFlag := flag.String("config", "", "Path to a MMMsg config file.")
	versionFlag := flag.Bool("version", false, "Print the version string.")
	codeBlock := flag.Bool("codeblock", false, "Surround the input with code block backticks")
	channelFlag := flag.String("channel", "", "Channel to post in, @username not yet supported")
	attachmentFlag := flag.String("attachment", "", "Upload and attach this file")

	// set a more verbose usage message.
	flag.Usage = func() {
		os.Stderr.WriteString(usageString)
		flag.PrintDefaults()
	}
	// parse them
	flag.Parse()

	// first do arg checking
	if *versionFlag {
		fmt.Println("Version: " + MMMsgVersion)
		fmt.Println(logoImage)
		fmt.Println("Project: https://github.com/AstromechZA/mmmsg")
		return nil
	}

	if *attachmentFlag != "" {
		if _, err := os.Stat(*attachmentFlag); os.IsNotExist(err) {
			return fmt.Errorf("Attachment file %v does not exist", *attachmentFlag)
		}
	}

	// now load the stdin
	inputBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	if len(inputBytes) == 0 {
		return fmt.Errorf("No stdin provided for message")
	}
	chars := []rune(string(inputBytes))
	characterLimit := 4000
	if *codeBlock {
		characterLimit -= 8
	}
	if len(chars) > characterLimit {
		chars = chars[:characterLimit]
	}
	stringContent := string(chars)
	if *codeBlock {
		stringContent = "```\n" + stringContent + "\n```"
	}

	// load and validate config
	configPath := (*configFlag)
	if configPath == "" {
		usr, _ := user.Current()
		configPath = filepath.Join(usr.HomeDir, ".config/mmmsg.json")
	}
	configPath, err = filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("Failed to identify config path: %v", err.Error())
	}

	// quick validate the config
	cfg, err := conf.Load(&configPath)
	if err != nil {
		return fmt.Errorf("Config failed to load: %v", err.Error())
	}
	err = conf.Validate(cfg)
	if err != nil {
		return fmt.Errorf("Config failed validation: %v", err.Error())
	}

	// override channel name from config if required
	targetChannelName := (*channelFlag)
	if targetChannelName == "" {
		targetChannelName = cfg.DefaultChannel
	}

	// now attempt connection stuff
	client := model.NewClient(cfg.MattermostAPIUrl)
	fmt.Printf("Connecting to Mattermost server at %v..\n", cfg.MattermostAPIUrl)
	serverProperties, apperr := client.GetPing()
	if apperr != nil {
		return fmt.Errorf("Failed to connect to Mattermost: %v", err.Error())
	}

	serverVersion := serverProperties["version"]
	fmt.Printf("Server at %v is running version %v\n", cfg.MattermostAPIUrl, serverVersion)
	fmt.Printf("Client version is %v\n", model.CurrentVersion)

	if !model.IsCurrentVersion(serverProperties["version"]) {
		if !model.IsPreviousVersionsSupported(serverVersion) {
			return fmt.Errorf("Server version %v is not supported", serverVersion)
		}
	}

	// login as bot
	fmt.Println("Logging in..")
	if _, err := client.Login(cfg.MattermostUser, cfg.MattermostPassword); err != nil {
		return fmt.Errorf("Failed to login to Mattermost: %v", err.Error())
	}

	// do initial load
	var initialLoad *model.InitialLoad
	fmt.Println("Loading initial data..")
	initialLoadResults, apperr := client.GetInitialLoad()
	if apperr != nil {
		return fmt.Errorf("Failed to get initial Mattermost data: %v", apperr.Error())
	}

	initialLoad = initialLoadResults.Data.(*model.InitialLoad)

	// find team info
	var botTeam *model.Team
	for _, team := range initialLoad.Teams {
		if team.Name == cfg.MattermostTeam {
			botTeam = team
			break
		}
	}
	if botTeam == nil {
		return fmt.Errorf("Bot does not appear to be a member of the team '%v'", cfg.MattermostTeam)
	}
	// set team
	client.SetTeamId(botTeam.Id)

	var targetChannel *model.Channel

	fmt.Println("Pulling list of channels..")
	channelsResult, apperr := client.GetChannels("")
	if apperr != nil {
		return fmt.Errorf("Failed to pull channel list: %v", apperr.Error())
	}
	channelList := *channelsResult.Data.(*model.ChannelList)

	// check if its a user channel
	if strings.HasPrefix(targetChannelName, "@") {
		// search of user with id
		username := targetChannelName[1:]
		fmt.Printf("Scanning for user by name '%v'..\n", username)
		userProfiles, apperr := client.GetProfiles(botTeam.Id, "")
		if apperr != nil {
			return fmt.Errorf("Failed to pull user profiles: %v", apperr.Error())
		}
		var targetUser *model.User
		userMap := userProfiles.Data.(map[string]*model.User)
		for _, u := range userMap {
			if u.Username == username {
				targetUser = u
				break
			}
		}

		if targetUser == nil {
			fmt.Println("No user by that username, maybe it is just a channel.")
		} else {

			fmt.Printf("Scanning for direct channel to user '%v'..\n", targetUser.Username)
			for _, channel := range channelList.Channels {
				if channel.Type == "D" {
					channelParts := strings.Split(channel.Name, "__")
					if channelParts[0] == targetUser.Id || channelParts[1] == targetUser.Id {
						targetChannel = channel
						break
					}
				}
			}

			if targetChannel == nil {
				fmt.Printf("Creating new Direct Message Channel..")
				directChannelResult, apperr := client.CreateDirectChannel(targetUser.Id)
				if apperr != nil {
					return fmt.Errorf("Failed to create direct message channel: %v", apperr.Error())
				}
				targetChannel = directChannelResult.Data.(*model.Channel)
			}
		}
	}

	// search for channel by name
	if targetChannel == nil {
		fmt.Printf("Scanning for channel '%v'..\n", targetChannelName)
		for _, channel := range channelList.Channels {
			if channel.Name == cfg.DefaultChannel {
				targetChannel = channel
				break
			}
		}
	}

	// error if still missing
	if targetChannel == nil {
		return fmt.Errorf("Failed to find channel with name '%v'", targetChannelName)
	}

	// prepare post
	post := &model.Post{}
	post.ChannelId = targetChannel.Id
	post.Message = stringContent

	// if upload is required, then do it
	if *attachmentFlag != "" {
		fmt.Printf("Attachment was specified, reading bytes from %v\n", *attachmentFlag)
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="files"; filename="%s"`,
				strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(filepath.Base(*attachmentFlag))))
		ct := mime.TypeByExtension(filepath.Ext(*attachmentFlag))
		if ct == "" {
			ct = "application/octet-stream"
		}
		h.Set("Content-Type", ct)
		part, _ := writer.CreatePart(h)
		file, err := os.Open(*attachmentFlag)
		if err != nil {
			return fmt.Errorf("Failed to read attachment file: %v", err.Error())
		}
		defer file.Close()
		_, err = io.Copy(part, file)
		if err != nil {
			return fmt.Errorf("Failed to read attachment file: %v", err.Error())
		}

		field, _ := writer.CreateFormField("channel_id")
		_, _ = field.Write([]byte(targetChannel.Id))
		err = writer.Close()
		if err != nil {
			return fmt.Errorf("Failed to build form body: %v", err.Error())
		}

		field, _ = writer.CreateFormField("channel_id")
		_, _ = field.Write([]byte(targetChannel.Id))
		err = writer.Close()
		if err != nil {
			return fmt.Errorf("Failed to build form body: %v", err.Error())
		}

		uploadResult, apperr := client.UploadPostAttachment(body.Bytes(), writer.FormDataContentType())
		if apperr != nil {
			return fmt.Errorf("Failed to upload file: %v", apperr.DetailedError)
		}
		uploadedFile := uploadResult.Data.(*model.FileUploadResponse)
		post.Filenames = uploadedFile.Filenames
	}

	fmt.Printf("Posting message of %v bytes..\n", len(stringContent))
	if _, err := client.CreatePost(post); err != nil {
		return fmt.Errorf("Failed to post message: %v", err.Error())
	}
	return nil
}

func main() {
	if err := mainInner(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
