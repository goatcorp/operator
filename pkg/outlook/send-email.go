package outlook

import (
	"fmt"
	"os"

	"github.com/jordan-wright/email"
)

func SendEmail(to, subject, body string) error {
	auth := LoginAuth(os.Getenv("OPERATOR_EMAIL"), os.Getenv("OPERATOR_PASSWORD"))
	e := email.NewEmail()
	e.To = []string{to}
	e.From = fmt.Sprintf("Caprine Operator <%s>", os.Getenv("OPERATOR_EMAIL"))
	e.Subject = subject
	e.HTML = []byte(body)

	err := e.Send(os.Getenv("OPERATOR_SMTP_SERVER"), auth)
	if err != nil {
		return err
	}

	return nil
}
