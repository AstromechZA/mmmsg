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
    	Channel to post in, @username not yet supported
  -codeblock
    	Surround the input with code block backticks
  -config string
    	Path to a MMMsg config file.
  -version
    	Print the version string.
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
