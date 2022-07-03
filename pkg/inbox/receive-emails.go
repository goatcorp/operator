package inbox

import (
	"hash/fnv"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx"
	"github.com/jprobinson/eazye"
	"github.com/microcosm-cc/bluemonday"
)

type ReceiveEmailsJob struct {
	Pool   *pgx.ConnPool
	Policy *bluemonday.Policy
}

func (j *ReceiveEmailsJob) Execute() {
	log.Println("Fetching unread operator emails")
	emails, err := getEmails(os.Getenv("OPERATOR_INBOX"))
	if err != nil {
		log.Printf("Failed to get incoming emails: %v\n", err)
	}

	junkEmails, err := getEmails(os.Getenv("OPERATOR_JUNK"))
	if err != nil {
		log.Printf("Failed to get incoming junk emails: %v\n", err)
	}

	emails = append(emails, junkEmails...)

	newReaders := make([]*ReaderInfo, 0)
	updatedReaders := make([]*ReaderInfo, 0)
	unsubscribers := make([]string, 0)
	for _, email := range emails {
		// Parse out the email information
		subjectCleaned := strings.TrimSpace(email.Subject)

		if strings.HasPrefix(subjectCleaned, "[op] subscribe") {
			r, err := ParseBody(email, *j.Policy)
			if err != nil {
				log.Printf("Failed to parse subscription email: %v\n", err)
				continue
			}

			// Validate reporting interval
			if r.ReportInterval.Minutes() <= 0 {
				log.Println("User attempted to set a reporting interval of 0 or less")
				continue
			}

			log.Println("Found new subscription email, adding to list")
			newReaders = append(newReaders, r)
		} else if strings.HasPrefix(subjectCleaned, "[op] update") {
			r, err := ParseBody(email, *j.Policy)
			if err != nil {
				log.Printf("Failed to parse update email: %v\n", err)
				continue
			}

			log.Println("Found new information update email, adding to list")
			updatedReaders = append(updatedReaders, r)
		} else if strings.HasPrefix(subjectCleaned, "[op] unsubscribe") {
			log.Println("Found new unsubscribe email, adding to list")
			unsubscribers = append(unsubscribers, email.From.Address)
		}
	}

	if len(newReaders) == 0 && len(updatedReaders) == 0 && len(unsubscribers) == 0 {
		log.Println("No unread operator emails found")
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
		log.Println("Processing new subscribers")
		saveSubscribers(readerConn, newReaders)
	}

	// Persist reader updates to the database
	if len(updatedReaders) > 0 {
		log.Println("Processing information update requests")
		saveUpdatedInfo(readerConn, updatedReaders)
	}

	// Delete unsubscribing readers from the database
	if len(unsubscribers) > 0 {
		log.Println("Processing unsubscribers")
		deleteUnsubscribers(readerConn, unsubscribers)
	}
}

func (j *ReceiveEmailsJob) Description() string {
	return "ReceiveEmailsJob"
}

func (j *ReceiveEmailsJob) Key() int {
	h := fnv.New32a()
	_, err := h.Write([]byte(j.Description()))
	if err != nil {
		log.Println(err)
		return -1
	}

	return int(h.Sum32())
}

func getEmails(mailbox string) ([]eazye.Email, error) {
	auth := eazye.MailboxInfo{
		Host:   os.Getenv("OPERATOR_IMAP_SERVER"),
		TLS:    true,
		User:   os.Getenv("OPERATOR_EMAIL"),
		Pwd:    os.Getenv("OPERATOR_PASSWORD"),
		Folder: mailbox,
	}

	emails, err := eazye.GetUnread(auth, true, false)
	if err != nil {
		return nil, err
	}

	return emails, nil
}
