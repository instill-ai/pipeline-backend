package email

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// Decide it temporarily
const EmailReadingDefaultCapacity = 100

type ReadEmailsInput struct {
	Search Search `json:"search"`
}

type Search struct {
	Mailbox            string `json:"mailbox"`
	SearchSubject      string `json:"search-subject,omitempty"`
	SearchFrom         string `json:"search-from,omitempty"`
	SearchTo           string `json:"search-to,omitempty"`
	Limit              int    `json:"limit,omitempty"`
	Date               string `json:"date,omitempty"`
	SearchEmailMessage string `json:"search-email-message,omitempty"`
}

type ReadEmailsOutput struct {
	Emails []Email `json:"emails"`
}

type Email struct {
	Date    string   `json:"date"`
	From    string   `json:"from"`
	To      []string `json:"to,omitempty"`
	Subject string   `json:"subject"`
	Message string   `json:"message,omitempty"`
}

func (e *execution) readEmails(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := ReadEmailsInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, err
	}

	setup := e.GetSetup()
	client, err := initIMAPClient(
		setup.GetFields()["server-address"].GetStringValue(),
		setup.GetFields()["server-port"].GetNumberValue(),
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	err = client.Login(
		setup.GetFields()["email-address"].GetStringValue(),
		setup.GetFields()["password"].GetStringValue(),
	).Wait()
	if err != nil {
		return nil, err
	}

	emails, err := fetchEmails(client, inputStruct.Search)
	if err != nil {
		return nil, err
	}

	if err := client.Logout().Wait(); err != nil {
		return nil, err
	}

	outputStruct := ReadEmailsOutput{
		Emails: emails,
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func initIMAPClient(serverAddress string, serverPort float64) (*imapclient.Client, error) {

	c, err := imapclient.DialTLS(fmt.Sprintf("%v:%v", serverAddress, serverPort), nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func fetchEmails(c *imapclient.Client, search Search) ([]Email, error) {

	selectedMbox, err := c.Select(search.Mailbox, nil).Wait()
	if err != nil {
		return nil, err
	}

	emails := []Email{}

	if selectedMbox.NumMessages == 0 {
		return emails, nil
	}

	if search.Limit == 0 {
		search.Limit = EmailReadingDefaultCapacity
	}
	limit := search.Limit

	// TODO: chuang8511, Research how to fetch emails by filter and concurrency.
	// It will be done before 2024-07-26.
	for i := selectedMbox.NumMessages; limit > 0; i-- {
		limit--
		email := Email{}
		var seqSet imap.SeqSet
		seqSet.AddNum(i)

		fetchOptions := &imap.FetchOptions{
			BodySection: []*imap.FetchItemBodySection{{}},
		}
		fetchCmd := c.Fetch(seqSet, fetchOptions)
		msg := fetchCmd.Next()
		var bodySection imapclient.FetchItemDataBodySection
		var ok bool
		for {
			item := msg.Next()
			if item == nil {
				break
			}
			bodySection, ok = item.(imapclient.FetchItemDataBodySection)
			if ok {
				break
			}
		}
		if !ok {
			return nil, fmt.Errorf("FETCH command did not return body section")
		}

		mr, err := mail.CreateReader(bodySection.Literal)
		if err != nil {
			return nil, fmt.Errorf("FETCH command did not return body section")
		}

		h := mr.Header
		setEnvelope(&email, h)

		if !checkEnvelopeCondition(email, search) {
			if err := fetchCmd.Close(); err != nil {
				return nil, err
			}
			continue
		}

		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			inlineHeader, ok := p.Header.(*mail.InlineHeader)
			if ok && !isHTMLType(inlineHeader) {
				b, _ := io.ReadAll(p.Body)
				email.Message += string(b)
			}
		}

		if !checkMessageCondition(email, search) {
			if err := fetchCmd.Close(); err != nil {
				return nil, err
			}
			continue
		}

		if err := fetchCmd.Close(); err != nil {
			return nil, err
		}

		emails = append(emails, email)
	}

	return emails, nil
}

func setEnvelope(email *Email, h mail.Header) {
	if date, err := h.Date(); err != nil {
		log.Println("Failed to get date")
	} else {
		email.Date = date.Format(time.DateTime)
	}
	if from, err := h.AddressList("From"); err != nil {
		log.Println("Failed to get from")
	} else {
		email.From = from[0].String()
	}
	if to, err := h.AddressList("To"); err != nil {
		log.Println("Failed to get to")
	} else {
		email.To = []string{}
		for _, t := range to {
			email.To = append(email.To, t.String())
		}
	}
	if subject, err := h.Text("Subject"); err != nil {
		log.Println("Failed to get subject")
	} else {
		email.Subject = subject
	}
}

func checkEnvelopeCondition(email Email, search Search) bool {
	if search.SearchSubject != "" {
		if !strings.Contains(email.Subject, search.SearchSubject) {
			return false
		}
	}
	if search.SearchFrom != "" {
		if email.From != search.SearchFrom {
			return false
		}
	}
	if search.SearchTo != "" {
		var found bool
		for _, t := range email.To {
			if t == search.SearchTo {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if search.Date != "" {
		if !strings.Contains(email.Date, search.Date) {
			return false
		}
	}
	return true
}

func checkMessageCondition(email Email, search Search) bool {
	if search.SearchEmailMessage != "" {
		if !strings.Contains(email.Message, search.SearchEmailMessage) {
			return false
		}
	}
	return true
}

func isHTMLType(inlineHeader *mail.InlineHeader) bool {
	return strings.Contains(inlineHeader.Get("Content-Type"), "text/html")
}
