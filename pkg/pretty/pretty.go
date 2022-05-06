package pretty

import (
	"html/template"
	"io"
)

type Plogon struct {
	Title     string
	URL       string
	Submitter string
}

func BuildTemplate(w io.Writer, plogons []*Plogon) error {
	t, err := template.ParseFiles("../../email-template.gohtml")
	if err != nil {
		return err
	}

	err = t.Execute(w, struct {
		Plogons []*Plogon
	}{Plogons: plogons})
	if err != nil {
		return err
	}

	return nil
}
