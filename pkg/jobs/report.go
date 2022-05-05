package jobs

import (
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
)

type ReportJob struct {
	Connection *pgx.Conn
}

func (j *ReportJob) Execute() {
	client := github.NewClient(nil)
	plogons, _, err := client.PullRequests.List(context.Background(), "goatcorp", "DalamudPlugins", &github.PullRequestListOptions{
		State: "open",
	})
	if err != nil {
		log.Printf("Request error: %v\n", err)
		return
	}

	plogonMsg := ""
	for _, plogon := range plogons {
		if plogon.Title != nil {
			plogonMsg += *plogon.Title
			plogonMsg += "\n"
		}
	}

	rows, err := j.getReadersToNotify()
	if err != nil {
		log.Printf("Unable to retrieve readers: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var readerEmail string
		var readerGithub string
		var readerLastSent time.Time
		rows.Scan(&readerEmail, &readerGithub, &readerLastSent)

		if rows.Err() != nil {
			log.Printf("Error occurred while processing readers: %v\n", rows.Err())
			return
		}

		log.Printf("Sending email to %s\n", readerEmail)
		err = sendEmail(readerEmail, plogonMsg)
		if err != nil {
			log.Printf("Unable to send mail: %v\n", err)
			return
		}
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

func (j *ReportJob) getReadersToNotify() (*pgx.Rows, error) {
	return j.Connection.Query(`
		SELECT Reader.email, Reader.github, max(Report.sent_time)
		FROM Reader
		LEFT JOIN Report
			ON Reader.id = Report.reader_id
		WHERE active
		GROUP BY Reader.id
		HAVING max(Report.sent_time) + Reader.report_interval <= now();
	`)
}

func sendEmail(to string, body string) error {
	auth := outlook.LoginAuth(os.Getenv("OPERATOR_EMAIL"), os.Getenv("OPERATOR_PASSWORD"))
	e := email.NewEmail()
	e.From = fmt.Sprintf("Caprine Operator <%s>", os.Getenv("OPERATOR_EMAIL"))
	e.To = []string{to}
	e.Text = []byte(body)

	err := e.Send(os.Getenv("OPERATOR_SMTP_SERVER"), auth)
	if err != nil {
		return err
	}

	return nil
}
