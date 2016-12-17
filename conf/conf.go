package conf

import (
    "fmt"
    "encoding/json"
    "io/ioutil"
)

// MMMsgConfig is the definition of the json config structure
type MMMsgConfig struct {
    MattermostAPIUrl string `json:"mattermost_api"`
    MattermostUser string `json:"mattermost_user"`
    MattermostPassword string `json:"mattermost_password"`
    MattermostTeam string `json:"mattermost_team"`
    DefaultChannel string `json:"default_channel"`
}

// Load the config information from the file on disk
func Load(path *string) (*MMMsgConfig, error) {

    // first read all bytes from file
    data, err := ioutil.ReadFile(*path)
    if err != nil {
        return nil, err
    }

    // now parse config object out
    var cfg MMMsgConfig
    err = json.Unmarshal(data, &cfg)
    if err != nil {
        return nil, err
    }

    // and return
    return &cfg, nil
}

// Validate a config that has already been loaded
func Validate(cfg *MMMsgConfig) error {
    if cfg.MattermostAPIUrl == "" {
        return fmt.Errorf("Config has no value for key 'mattermost_api'")
    }
    if cfg.MattermostUser == "" {
        return fmt.Errorf("Config has no value for key 'mattermost_user'")
    }
    if cfg.MattermostPassword == "" {
        return fmt.Errorf("Config has no value for key 'mattermost_password'")
    }
    if cfg.MattermostTeam == "" {
        return fmt.Errorf("Config has no value for key 'mattermost_team'")
    }
    return nil
}
