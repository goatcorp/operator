package inbox

import (
	"bytes"
	"io"
	"log"
	"text/template"
	"time"

	"github.com/jackc/pgx"
	"github.com/karashiiro/operator/pkg/html"
	"github.com/karashiiro/operator/pkg/outlook"
)

func saveUpdatedInfo(conn *pgx.Conn, readers []*ReaderInfo) {
	for _, r := range readers {
		if r.GitHubSet {
			_, err := updateGitHub(conn, r.GitHub)
			if err != nil {
				log.Printf("Failed to update reader GitHub: %v\n", err)
				continue
			}
		}

		if r.ReportInterval.Minutes() > 0 {
			_, err := updateReportInterval(conn, r.ReportInterval)
			if err != nil {
				log.Printf("Failed to update reader report interval: %v\n", err)
				continue
			}
		}

		log.Printf("Sending update confirmation email to %s\n", r.Email)

		var updateMessage bytes.Buffer
		err := buildUpdateTemplate(&updateMessage, r.ReportInterval)
		if err != nil {
			log.Printf("Failed to build update template: %v\n", err)
		}

		err = outlook.SendEmail(r.Email, "Information updated", updateMessage.String())
		if err != nil {
			log.Printf("Unable to send mail: %v\n", err)
			continue
		}

		log.Printf("Updated reader %s\n", r.Email)
	}
}

func buildUpdateTemplate(w io.Writer, interval time.Duration) error {
	t, err := template.ParseFS(html.Files, "confirm-update.gohtml")
	if err != nil {
		return err
	}

	err = t.Execute(w, struct {
		Interval time.Duration
	}{
		Interval: interval,
	})
	if err != nil {
		return err
	}

	return nil
}

func updateGitHub(conn *pgx.Conn, gh string) (int64, error) {
	var github *string
	if gh != "" {
		github = &gh
	}

	t, err := conn.Exec(`
		UPDATE Reader SET github = $1;
	`, github)
	if err != nil {
		return 0, err
	}

	return t.RowsAffected(), nil
}

func updateReportInterval(conn *pgx.Conn, interval time.Duration) (int64, error) {
	t, err := conn.Exec(`
		UPDATE Reader SET report_interval = $1;
	`, interval)
	if err != nil {
		return 0, err
	}

	return t.RowsAffected(), nil
}
