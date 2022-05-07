package inbox

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx"
	"github.com/jprobinson/eazye"
	"github.com/karashiiro/operator/pkg/html"
	"github.com/karashiiro/operator/pkg/outlook"
)

var githubPattern = regexp.MustCompile(`(?i)github:\s*(?P<github>\S*)`)
var intervalPattern = regexp.MustCompile(`(?i)interval:\s*(?P<interval>\S*)`)

func (j *ReceiveEmailsJob) subscribe(email eazye.Email) (*newReader, error) {
	r := &newReader{
		Email: j.Policy.Sanitize(email.From.Address),
	}

	bodyLines := strings.Split(string(email.Text), "\n")
	for _, line := range bodyLines {
		lineCleaned := strings.TrimSpace(line)

		// Parse their optionally-provided GitHub username
		githubMatches := githubPattern.FindStringSubmatch(lineCleaned)
		if len(githubMatches) != 0 {
			github := githubMatches[githubPattern.SubexpIndex("github")]
			r.GitHub = j.Policy.Sanitize(github)
			continue
		}

		// Parse their requested reporting interval
		intervalMatches := intervalPattern.FindStringSubmatch(lineCleaned)
		if len(intervalMatches) != 0 {
			interval, err := time.ParseDuration(intervalMatches[intervalPattern.SubexpIndex("interval")])
			if err == nil {
				r.ReportInterval = interval
				continue
			}
		}
	}

	// Validate reporting interval
	if r.ReportInterval.Minutes() <= 0 {
		return nil, fmt.Errorf("user attempted to set a reporting interval of 0 or less")
	}

	return r, nil
}

func saveSubscribers(conn *pgx.Conn, readers []*newReader) {
	for _, r := range readers {
		_, err := storeReader(conn, r)
		if err != nil {
			log.Printf("Failed to add new reader: %v\n", err)
			continue
		}

		log.Printf("Sending subscription confirmation email to %s\n", r.Email)

		var subscribeMessage bytes.Buffer
		err = buildSubscribeTemplate(&subscribeMessage, r.ReportInterval)
		if err != nil {
			log.Printf("Failed to build subscribe template: %v\n", err)
		}

		err = outlook.SendEmail(r.Email, "Subscription confirmed", subscribeMessage.String())
		if err != nil {
			log.Printf("Unable to send mail: %v\n", err)
			continue
		}

		log.Printf("Added reader %s\n", r.Email)
	}
}

func buildSubscribeTemplate(w io.Writer, interval time.Duration) error {
	t, err := template.ParseFS(html.Files, "confirm-subscribe.gohtml")
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

func storeReader(conn *pgx.Conn, r *newReader) (int64, error) {
	t, err := conn.Exec(`
		INSERT INTO Reader (email, github, report_interval, active)
		VALUES
			($1, $2, $3, TRUE)
	`, r.Email, r.GitHub, r.ReportInterval)
	if err != nil {
		return 0, err
	}

	return t.RowsAffected(), nil
}
