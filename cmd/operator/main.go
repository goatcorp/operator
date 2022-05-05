package main

import (
	"time"

	"github.com/karashiiro/operator/pkg/jobs"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	sched := quartz.NewStdScheduler()
	sched.Start()
	trigger := quartz.NewSimpleTrigger(time.Second)
	job := jobs.ReportJob{}
	sched.ScheduleJob(&job, trigger)
	time.Sleep(3 * time.Second)
	sched.Stop()
}
