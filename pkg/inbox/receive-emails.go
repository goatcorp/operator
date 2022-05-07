package inbox

import (
	"hash/fnv"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx"
	"github.com/jprobinson/eazye"
	"github.com/microcosm-cc/bluemonday"
)

type ReceiveEmailsJob struct {
	Pool   *pgx.ConnPool
	Policy *bluemonday.Policy
}

type newReader struct {
	Email          string
	GitHub         string
	ReportInterval time.Duration
}

func (j *ReceiveEmailsJob) Execute() {
	auth := eazye.MailboxInfo{
		Host:   os.Getenv("OPERATOR_IMAP_SERVER"),
		TLS:    true,
		User:   os.Getenv("OPERATOR_EMAIL"),
		Pwd:    os.Getenv("OPERATOR_PASSWORD"),
		Folder: os.Getenv("OPERATOR_INBOX"),
	}

	emails, err := eazye.GetUnread(auth, true, false)
	if err != nil {
		log.Printf("Failed to get incoming emails: %v\n", err)
	}

	newReaders := make([]*newReader, 0)
	unsubscribers := make([]string, 0)
	for _, email := range emails {
		// Parse out the email information
		subjectCleaned := strings.TrimSpace(email.Subject)

		if strings.HasPrefix(subjectCleaned, "[op] subscribe") {
			r, err := j.subscribe(email)
			if err != nil {
				log.Printf("Failed to parse subscription email: %v\n", err)
			}

			newReaders = append(newReaders, r)
		} else if strings.HasPrefix(subjectCleaned, "[op] unsubscribe") {
			unsubscribers = append(unsubscribers, email.From.Address)
		}
	}

	if len(newReaders) == 0 && len(unsubscribers) == 0 {
		return
	}

	readerConn, err := j.Pool.Acquire()
	if err != nil {
		log.Printf("Failed to acquire database connection: %v\n", err)
		return
	}
	defer j.Pool.Release(readerConn)

	// Save new readers to the database
	if len(newReaders) > 0 {
		saveSubscribers(readerConn, newReaders)
	}

	// Delete unsubscribing readers from the database
	if len(unsubscribers) > 0 {
		deleteUnsubscribers(readerConn, unsubscribers)
	}
}

func (j *ReceiveEmailsJob) Description() string {
	return "ReceiveEmailsJob"
}

func (j *ReceiveEmailsJob) Key() int {
	h := fnv.New32a()
	h.Write([]byte(j.Description()))
	return int(h.Sum32())
}
