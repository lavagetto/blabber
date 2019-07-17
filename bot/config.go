package bot

import (
	"encoding/json"
	"fmt"
	"os"
)

// Configuration holds all the configuration of
// the bot
type Configuration struct {
	ServerName string   `json:"server"`
	ServerPort uint     `json:"port"`
	UseTLS     bool     `json:"use_tls"`
	UseSASL    bool     `json:"use_sasl"`
	NickName   string   `json:"nick"`
	Password   string   `json:"password"`
	Channels   []string `json:"channels"`
	DbDsn      string   `json:"db_dsn"`
	Admins     []string `json:"admins"`
}

// GetConfig returns a configuration object
// from reading a properly formatted json file
func GetConfig(fileName string) (*Configuration, error) {
	config := Configuration{
		ServerName: "irc.freenode.net",
		ServerPort: 6697,
		UseTLS:     true,
		UseSASL:    true,
		NickName:   "BlabberBot",
		Channels:   []string{"#somechannel"},
		DbDsn:      "sqlite3://file:blabber.db?cache=shared",
	}
	if fileName == "" {
		return &config, nil
	}
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, err
}

// GetServerString gives you a host:port string of the server to connect to.
func (c *Configuration) GetServerString() string {
	return fmt.Sprintf("%s:%d", c.ServerName, c.ServerPort)
}
