package mailer

import (
	"bytes"
	"embed"
	"html/template"

	"github.com/wneessen/go-mail"
)

//go:embed "templates"
var tmpl embed.FS

type Mailer struct {
	client *mail.Client
	sender string
}

func NewMailer(host string, port int, username, password, sender string) (Mailer, error) {
	client, err := mail.NewClient(host,
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(username),
		mail.WithPassword(password),
	)
	if err != nil {
		return Mailer{}, err
	}
	return Mailer{
		client: client,
		sender: sender,
	}, nil
}

func (m Mailer) Send(recipient, tmplFile string, data any) error {
	ts, err := template.ParseFS(tmpl, "templates/"+tmplFile)
	if err != nil {
		return err
	}

	var subject bytes.Buffer

	err = ts.ExecuteTemplate(&subject, "subject", data)
	if err != nil {
		return err
	}

	var plainBody bytes.Buffer

	err = ts.ExecuteTemplate(&plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	var htmlBody bytes.Buffer
	err = ts.ExecuteTemplate(&htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	msg := mail.NewMsg()
	err = msg.From(m.sender)
	if err != nil {
		return err
	}
	err = msg.To(recipient)
	if err != nil {
		return err
	}
	msg.Subject(subject.String())
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())

	return m.client.DialAndSend(msg)
}
