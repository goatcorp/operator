package outlook

import (
	"fmt"
	"net/smtp"
)

// https://gist.github.com/homme/22b457eb054a07e7b2fb
type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	req := string(fromServer)
	if more {
		switch req {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, fmt.Errorf("unknown server request field %s", req)
		}
	}
	return nil, nil
}
