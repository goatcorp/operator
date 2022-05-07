package reports

import (
	"html/template"
	"io"
	"time"

	"github.com/karashiiro/operator/pkg/html"
)

type PlogonLabel struct {
	Name  string
	Color string
}

type Plogon struct {
	Title     string
	URL       string
	Labels    []*PlogonLabel
	Submitter string
	Updated   time.Time
}

func BuildTemplate(w io.Writer, plogons []*Plogon) error {
	t, err := template.New("report.gohtml").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format(time.RFC822)
		},
	}).ParseFS(html.Files, "report.gohtml")
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
