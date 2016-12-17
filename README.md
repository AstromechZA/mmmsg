# `mmmsg` Mattermost message sender

A quick cli tool to connect to a Mattermost chat server (v3.4.0) and send a 
message. 

```
mmmsg is a slim binary for pushing a mattermost message to
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

  -attachment string
    	Upload and attach this file
  -channel string
    	Channel to post in, @username for direct message
  -codeblock
    	Surround the input with code block backticks
  -config string
    	Path to a MMMsg config file (default: $HOME/.config/mmmsg.json)
  -version
    	Print the version string
```

Example:

```
$ echo "my prepared message content" | ./mmmsg --channel "alerts-channel" --attachment "alert.json"
Connecting to Mattermost server at http://example-server.com..
Server at example-server.com is running version 3.4.0
Client version is 3.4.0
Logging in..
Loading initial data..
Pulling list of channels..
Scanning for channel 'alerts-channel'..
Attachment was specified, reading bytes from alert.json
Posting message of 27 bytes..
```

### Why not just use the built in incoming webhooks and ignore the complicated
direct API calls?

This binary is to serve a more generic purpose, it can send to any user or 
channel without needing a webhook set up. It also allows you to attach a file
to the message which webhooks don't really allow yet.

For example, user submits a build against a CI server. The CI server can use 
this tool to send a message directly to the user without needing explicit 
webhooks set up for each user.

It's also for simple cli usage by a user from their terminal.

### API compatibility

I tried to build it using v3.5.0 talking to our v3.4.0 server, but it had some
troubles due to non-backward compatible API changes, so for now I'm tracking
our server at work which for now is at v3.4.0 :smile: 
