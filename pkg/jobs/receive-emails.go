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
	unsubscribers := make([]string, 0)
	for _, email := range emails {
		// Parse out the email information
		subjectCleaned := strings.TrimSpace(email.Subject)

		if strings.HasPrefix(subjectCleaned, "[op] subscribe") {
			r, err := j.subscribe(email)
			if err != nil {
				log.Printf("Failed to parse subscription email: %v\n", err)
			}

			newReaders = append(newReaders, r)
		} else if strings.HasPrefix(subjectCleaned, "[op] unsubscribe") {
			unsubscribers = append(unsubscribers, email.From.Address)
		}
	}

	if len(newReaders) == 0 && len(unsubscribers) == 0 {
		return
	}

	readerConn, err := j.Pool.Acquire()
	if err != nil {
		log.Printf("Failed to acquire database connection: %v\n", err)
		return
	}
	defer j.Pool.Release(readerConn)

	// Save new readers to the database
	if len(newReaders) > 0 {
		saveSubscribers(readerConn, newReaders)
	}

	// Delete unsubscribing readers from the database
	if len(unsubscribers) > 0 {
		deleteUnsubscribers(readerConn, unsubscribers)
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
	if err != nil {
		return 0, err
	}

	return t.RowsAffected(), nil
}

func deleteUnsubscribers(conn *pgx.Conn, unsubscribers []string) {
	for _, us := range unsubscribers {
		_, err := deleteReader(conn, us)
		if err != nil {
			log.Printf("Failed to delete reader: %v\n", err)
			continue
		}

		var unsubscribeMessage bytes.Buffer
		err = buildUnsubscribeTemplate(&unsubscribeMessage)
		if err != nil {
			log.Printf("Failed to build unsubscribe template: %v\n", err)
			continue
		}

		err = outlook.SendEmail(us, "Unsubscribe confirmed", unsubscribeMessage.String())
		if err != nil {
			log.Printf("Unable to send mail: %v\n", err)
			continue
		}

		log.Printf("Deleted reader %s\n", us)
	}
}

func deleteReader(conn *pgx.Conn, addr string) (int64, error) {
	t1, err := conn.Exec("DELETE FROM Report WHERE reader_id = (SELECT id FROM Reader WHERE email = $1);", addr)
	if err != nil {
		return 0, err
	}

	t2, err := conn.Exec("DELETE FROM Reader WHERE email = $1;", addr)
	if err != nil {
		return t1.RowsAffected(), err
	}

	return t1.RowsAffected() + t2.RowsAffected(), nil
}

func buildUnsubscribeTemplate(w io.Writer) error {
	t, err := template.ParseFiles("./templates/confirm-unsubscribe.gohtml")
	if err != nil {
		return err
	}

	err = t.Execute(w, struct{}{})
	if err != nil {
		return err
	}

	return nil
}
