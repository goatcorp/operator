package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx"
	"github.com/karashiiro/operator/pkg/db"
	"github.com/karashiiro/operator/pkg/inbox"
	"github.com/karashiiro/operator/pkg/reports"
	"github.com/karashiiro/operator/pkg/sql"
	"github.com/microcosm-cc/bluemonday"
	"github.com/reugn/go-quartz/quartz"
)

func applyMigrations(pool *pgx.ConnPool) {
	conn, err := pool.Acquire()
	if err != nil {
		log.Printf("Failed to acquire database connection: %v\n", err)
		os.Exit(1)
	}
	defer pool.Release(conn)

	err = db.ApplyMigrations(conn, sql.Files)
	if err != nil {
		log.Printf("Failed to apply migrations: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	// Create the database connection pool
	config := &pgx.ConnConfig{
		User:     "operator",
		Password: "operator",
		Database: "operator",
	}

	postgresHost := os.Getenv("OPERATOR_POSTGRES")
	if postgresHost != "" {
		log.Printf("Using PostgreSQL host %s\n", postgresHost)
		config.Host = postgresHost
	}

	pool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     *config,
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

	// Apply the database migrations
	applyMigrations(pool)

	// Start the job scheduler
	sched := quartz.NewStdScheduler()
	sched.Start()

	// Schedule the report job
	reportTrigger := quartz.NewSimpleTrigger(2 * time.Minute)
	reportJob := reports.ReportJob{Pool: pool}
	sched.ScheduleJob(&reportJob, reportTrigger)

	// Schedule the email-checking job
	receiveTrigger := quartz.NewSimpleTrigger(time.Minute)
	receiveJob := inbox.ReceiveEmailsJob{
		Pool:   pool,
		Policy: bluemonday.UGCPolicy(),
	}
	sched.ScheduleJob(&receiveJob, receiveTrigger)

	// Block until SIGINT or SIGTERM is received
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	// Shutdown
	sched.Stop()
}
