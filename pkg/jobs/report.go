package jobs

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"os"

	"github.com/google/go-github/v44/github"
	"github.com/jackc/pgx"
	"github.com/jordan-wright/email"
	"github.com/karashiiro/operator/pkg/outlook"
)

type ReportJob struct{}

func (j *ReportJob) Execute() {
	client := github.NewClient(nil)
	plogons, _, err := client.PullRequests.List(context.Background(), "goatcorp", "DalamudPlugins", &github.PullRequestListOptions{
		State: "open",
	})

	if err != nil {
		log.Println("Request error")
		return
	}

	plogonMsg := ""
	for _, plogon := range plogons {
		if plogon.Title != nil {
			plogonMsg += *plogon.Title
			plogonMsg += "\n"
		}
	}

	conn, err := pgx.Connect(pgx.ConnConfig{
		User:     "operator",
		Password: "operator",
		Database: "operator",
	})
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	log.Println("Sending email")
	a := outlook.LoginAuth(os.Getenv("OPERATOR_EMAIL"), os.Getenv("OPERATOR_PASSWORD"))
	from := fmt.Sprintf("Caprine Operator <%s>", os.Getenv("OPERATOR_EMAIL"))
	to := []string{""}
	msg := []byte(plogonMsg)

	e := email.NewEmail()
	e.From = from
	e.To = to
	e.Text = msg
	err = e.Send(os.Getenv("OPERATOR_SMTP_SERVER"), a)
	if err != nil {
		log.Printf("Unable to send mail: %v\n", err)
		os.Exit(1)
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
