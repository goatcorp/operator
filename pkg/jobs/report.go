package jobs

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"time"

	"github.com/google/go-github/v44/github"
	"github.com/jackc/pgx"
	"github.com/jordan-wright/email"
	"github.com/karashiiro/operator/pkg/outlook"
	"github.com/karashiiro/operator/pkg/pretty"
)

type ReportJob struct {
	Pool *pgx.ConnPool
}

func (j *ReportJob) Execute() {
	// Retrieve all open pull requests
	client := github.NewClient(nil)
	plogons, _, err := client.PullRequests.List(context.Background(), "goatcorp", "DalamudPlugins", &github.PullRequestListOptions{
		State: "open",
	})
	if err != nil {
		log.Printf("Request error: %v\n", err)
		return
	}

	// Make the plogons :dognosepretty:
	plogonsPretty := make([]*pretty.Plogon, len(plogons))
	for i, plogon := range plogons {
		plogonsPretty[i] = &pretty.Plogon{
			Title:     plogon.GetTitle(),
			URL:       plogon.GetURL(),
			Submitter: plogon.User.GetLogin(),
		}
	}

	// Retrieve all of the active report readers
	readerConn, err := j.Pool.Acquire()
	if err != nil {
		log.Printf("Failed to acquire database connection: %v\n", err)
		return
	}
	defer readerConn.Close()

	rows, err := getReadersToNotify(readerConn)
	if err != nil {
		log.Printf("Unable to retrieve readers: %v\n", err)
		return
	}
	defer rows.Close()

	// Open a connection to handle inserting records into the report table
	// while we're iterating the results of the reader query
	reportConn, err := j.Pool.Acquire()
	if err != nil {
		log.Printf("Failed to acquire database connection: %v\n", err)
		return
	}
	defer reportConn.Close()

	for rows.Next() {
		var readerId int
		var readerEmail string
		var readerGithub string
		var readerLastSent time.Time
		err := rows.Scan(&readerId, &readerEmail, &readerGithub, &readerLastSent)
		if err != nil {
			log.Printf("Unable to scan reader row: %v\n", err)
			continue
		}

		var readerMessage bytes.Buffer
		err = pretty.BuildTemplate(&readerMessage, plogonsPretty)
		if err != nil {
			log.Printf("Failed to build template: %v\n", err)
			continue
		}

		log.Printf("Sending email to %s\n", readerEmail)
		err = sendEmail(readerEmail, "Dalamud Plugin Pull Requests", readerMessage.String())
		if err != nil {
			log.Printf("Unable to send mail: %v\n", err)
			continue
		}

		_, err = storeReportLog(reportConn, readerId)
		if err != nil {
			log.Printf("Unable to store report log: %v\n", err)
			continue
		}
	}

	if rows.Err() != nil {
		log.Printf("Error occurred while processing readers: %v\n", rows.Err())
		return
	}
}

func (j *ReportJob) Description() string {
	return "ReportJob"
}

func (j *ReportJob) Key() int {
	h := fnv.New32a()
	h.Write([]byte(j.Description()))
	return int(h.Sum32())
}

func getReadersToNotify(conn *pgx.Conn) (*pgx.Rows, error) {
	return conn.Query(`
		SELECT Reader.id, Reader.email, Reader.github, max(Report.sent_time)
		FROM Reader
		LEFT JOIN Report
			ON Reader.id = Report.reader_id
		WHERE active
		GROUP BY Reader.id
		HAVING max(Report.sent_time) + Reader.report_interval <= now();
	`)
}

func storeReportLog(conn *pgx.Conn, readerId int) (int64, error) {
	tag, err := conn.Exec(`
		INSERT INTO Report (sent_time, reader_id)
		VALUES
			(now(), $1);
	`, readerId)
	return tag.RowsAffected(), err
}

func sendEmail(to, subject, body string) error {
	auth := outlook.LoginAuth(os.Getenv("OPERATOR_EMAIL"), os.Getenv("OPERATOR_PASSWORD"))
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
