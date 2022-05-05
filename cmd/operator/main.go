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

func applyMigrations(pool *pgx.ConnPool) {
	conn, err := pool.Acquire()
	if err != nil {
		log.Printf("Failed to acquire database connection: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	err = db.ApplyMigrations(conn, "../../sql")
	if err != nil {
		log.Printf("Failed to apply migrations: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	pool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			User:     "operator",
			Password: "operator",
			Database: "operator",
		},
		MaxConnections: 4,
		AfterConnect: func(c *pgx.Conn) error {
			log.Println("Database connection opened")
			return nil
		},
	})
	if err != nil {
		log.Printf("Unable to create database connection pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	applyMigrations(pool)

	sched := quartz.NewStdScheduler()
	sched.Start()
	trigger := quartz.NewRunOnceTrigger(time.Second)
	job := jobs.ReportJob{Pool: pool}
	sched.ScheduleJob(&job, trigger)
	time.Sleep(5 * time.Second)
	sched.Stop()
}
