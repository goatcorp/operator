package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/karashiiro/operator/pkg/html"
	"github.com/karashiiro/operator/pkg/reports"
)

func reportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	t, err := template.New("report.gohtml").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format(time.RFC822)
		},
	}).ParseFS(html.Files, "report.gohtml", "report-problems.gohtml")
	if err != nil {
		_, err := w.Write([]byte(fmt.Sprintf("%v\n", err)))
		if err != nil {
			log.Println(err)
		}

		return
	}

	reportTemplates, err := reports.GetPlogonReportTemplates()
	if err != nil {
		_, err := w.Write([]byte(fmt.Sprintf("%v\n", err)))
		if err != nil {
			log.Println(err)
		}

		return
	}

	err = t.Execute(w, struct {
		PlogonStates []*reports.ReportTemplate
	}{
		PlogonStates: reportTemplates,
	})
	if err != nil {
		_, err := w.Write([]byte(fmt.Sprintf("%v\n", err)))
		if err != nil {
			log.Println(err)
		}

		return
	}
}

func main() {
	http.HandleFunc("/report", reportHandler)
	http.ListenAndServe(":9000", nil)
}
