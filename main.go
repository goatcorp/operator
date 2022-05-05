package main

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"net/smtp"
	"os"
	"time"

	"github.com/google/go-github/v44/github"
	"github.com/jackc/pgx"
	"github.com/jordan-wright/email"
	"github.com/reugn/go-quartz/quartz"
)

// https://gist.github.com/homme/22b457eb054a07e7b2fb
type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unknown fromServer")
		}
	}
	return nil, nil
}

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

	for _, plogon := range plogons {
		if plogon.Title != nil {
			log.Println(*plogon.Title)
		} else {
			log.Println()
		}
	}

	conn, err := pgx.Connect(pgx.ConnConfig{
		User:     "operator",
		Password: "operator",
		Database: "operator",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	a := LoginAuth(os.Getenv("OPERATOR_EMAIL"), os.Getenv("OPERATOR_PASSWORD"))
	from := fmt.Sprintf("Caprine Operator <%s>", os.Getenv("OPERATOR_EMAIL"))
	to := []string{""}
	msg := []byte("test")

	e := email.NewEmail()
	e.From = from
	e.To = to
	e.Text = msg
	err = e.Send(os.Getenv("OPERATOR_SMTP_SERVER"), a)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to send mail: %v\n", err)
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

func main() {
	sched := quartz.NewStdScheduler()
	sched.Start()
	trigger := quartz.NewSimpleTrigger(time.Second)
	job := ReportJob{}
	sched.ScheduleJob(&job, trigger)
	time.Sleep(3 * time.Second)
	sched.Stop()
}
