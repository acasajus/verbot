package main

import (
	"log"

	"os"
	"verbot/bot"
	"verbot/constants"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func parseConf() (bot.Conf, error) {
	var confFile *string = pflag.StringP("conf", "c", "verbot.toml", "Configuration file")
	var version *bool = pflag.BoolP("version", "v", false, "Show version")

	pflag.Parse()

	if *version {
		log.Printf("Version is: %s", constants.VERSION)
		os.Exit(0)
	}
	conf := bot.Conf{}

	viper.SetConfigFile(*confFile)
	viper.SetEnvPrefix("VERBOT")
	err := viper.ReadInConfig()
	if err != nil {
		return conf, err
	}
	viper.SetDefault("Url", "http://localhost:8065")
	viper.SetDefault("Login", "")
	viper.SetDefault("Password", "")
	viper.SetDefault("Team", "verbio")
	viper.SetDefault("DebugChannel", "verbot-debug")
	viper.SetDefault("Bot.Username", "verbot")
	viper.SetDefault("Bot.FirstName", "VerBot")
	viper.SetDefault("Bot.LastName", "")
	err = viper.Unmarshal(&conf)
	if err != nil {
		return conf, err
	}
	if err = conf.Validate(); err != nil {
		return conf, errors.Wrap(err, "Invalid configuration")
	}
	return conf, nil
}

func main() {
	conf, err := parseConf()
	if err != nil {
		log.Fatal(err)
	}
	bot, err := bot.Connect(conf)
	if err != nil {
		log.Fatal(err)
	}
	bot.Wait()
}
