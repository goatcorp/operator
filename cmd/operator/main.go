package main

import (
	"log"
	"os"
	"time"

	"github.com/jackc/pgx"
	"github.com/karashiiro/operator/pkg/db"
	"github.com/karashiiro/operator/pkg/jobs"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
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

	err = db.ApplyMigrations(conn, "../../sql")
	if err != nil {
		log.Printf("Failed to apply migrations: %v\n", err)
		os.Exit(1)
	}

	sched := quartz.NewStdScheduler()
	sched.Start()
	trigger := quartz.NewRunOnceTrigger(time.Second)
	job := jobs.ReportJob{Connection: conn}
	sched.ScheduleJob(&job, trigger)
	time.Sleep(5 * time.Second)
	sched.Stop()
}
