package mailer

import "gymmaster-api/internal/config"

func MailCfgForTest() config.MailCfg {
	return config.MailCfg{Host: "localhost", Port: 1, From: "t@t.vn"}
}
