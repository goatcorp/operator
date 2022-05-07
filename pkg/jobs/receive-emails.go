package jobs

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx"
	"github.com/jprobinson/eazye"
	"github.com/karashiiro/operator/pkg/outlook"
	"github.com/microcosm-cc/bluemonday"
)

var githubPattern = regexp.MustCompile(`(?i)github:\s*(?P<github>\S*)`)
var intervalPattern = regexp.MustCompile(`(?i)interval:\s*(?P<interval>\S*)`)

type ReceiveEmailsJob struct {
	Pool   *pgx.ConnPool
	Policy *bluemonday.Policy
}

type newReader struct {
	Email          string
	GitHub         string
	ReportInterval time.Duration
}

func (j *ReceiveEmailsJob) Execute() {
	auth := eazye.MailboxInfo{
		Host:   os.Getenv("OPERATOR_IMAP_SERVER"),
		TLS:    true,
		User:   os.Getenv("OPERATOR_EMAIL"),
		Pwd:    os.Getenv("OPERATOR_PASSWORD"),
		Folder: os.Getenv("OPERATOR_INBOX"),
	}

	emails, err := eazye.GetUnread(auth, true, false)
	if err != nil {
		log.Printf("Failed to get incoming emails: %v\n", err)
	}

	newReaders := make([]*newReader, 0)
	for _, email := range emails {
		// Parse out the email information
		subjectCleaned := strings.TrimSpace(email.Subject)
		if strings.HasPrefix(subjectCleaned, "[op] subscribe") {
			r, err := j.subscribe(email)
			if err != nil {
				log.Printf("Failed to parse subscription email: %v\n", err)
			}

			newReaders = append(newReaders, r)
		}
	}

	// Save new readers to the database
	if len(newReaders) > 0 {
		readerConn, err := j.Pool.Acquire()
		if err != nil {
			log.Printf("Failed to acquire database connection: %v\n", err)
			return
		}
		defer j.Pool.Release(readerConn)

		for _, r := range newReaders {
			_, err := storeReader(readerConn, r)
			if err != nil {
				log.Printf("Failed to add new reader: %v\n", err)
			}

			log.Printf("Sending subscription confirmation email to %s\n", r.Email)

			var subscribeMessage bytes.Buffer
			buildSubscriptionTemplate(&subscribeMessage, r.ReportInterval)

			err = outlook.SendEmail(r.Email, "Subscription confirmed", subscribeMessage.String())
			if err != nil {
				log.Printf("Unable to send mail: %v\n", err)
				continue
			}

			log.Printf("Added reader %s\n", r.Email)
		}
	}
}

func (j *ReceiveEmailsJob) Description() string {
	return "ReceiveEmailsJob"
}

func (j *ReceiveEmailsJob) Key() int {
	h := fnv.New32a()
	h.Write([]byte(j.Description()))
	return int(h.Sum32())
}

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

func buildSubscriptionTemplate(w io.Writer, interval time.Duration) error {
	t, err := template.ParseFiles("./templates/confirm-subscribe.gohtml")
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
	return t.RowsAffected(), err
}
