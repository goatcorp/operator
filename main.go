package main

import (
	"hash/fnv"
	"log"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

type ReportJob struct{}

func (j *ReportJob) Execute() {
	log.Println("hit")
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
	time.Sleep(5 * time.Second)
	sched.Stop()
}
