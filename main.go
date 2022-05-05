package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"time"

	"github.com/google/go-github/v44/github"
	"github.com/jackc/pgx"
	"github.com/reugn/go-quartz/quartz"
)

type ReportJob struct{}

func (j *ReportJob) Execute() {
	client := github.NewClient(nil)
	repos, _, err := client.PullRequests.List(context.Background(), "goatcorp", "DalamudPlugins", &github.PullRequestListOptions{
		State: "open",
	})

	if err != nil {
		log.Println("Request error")
		return
	}

	for _, repo := range repos {
		if repo.Title != nil {
			log.Println(*repo.Title)
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
