package bot

import "errors"

type Conf struct {
	Url          string
	Login        string
	Password     string
	Team         string
	DebugChannel string
	Bot          BotNameConf
}

func (c Conf) Validate() error {
	if len(c.Url) == 0 {
		return errors.New("Missing Url")
	}
	if len(c.Login) == 0 {
		return errors.New("Missing Login")
	}
	if len(c.Password) == 0 {
		return errors.New("Missing Password")
	}
	if len(c.Team) == 0 {
		return errors.New("Missing Team")
	}
	return nil
}

type BotNameConf struct {
	Username string
	First    string
	Last     string
}
