package email

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"
	"cmp"
	"reflect"
	"sort"
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

func fetchEmail(c *imapclient.Client, search Search, seqSet imap.SeqSet, fetchOptions imap.FetchOptions, ch chan Email, errChan chan<- error) {
	email := Email{}
	fetchCmd := c.Fetch(seqSet, &fetchOptions)
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
		errChan <- fmt.Errorf("FETCH command did not return body section")
		return
	}

	mr, err := mail.CreateReader(bodySection.Literal)

	if err != nil {
		errChan <- fmt.Errorf("FETCH command did not return body section")
		return
	}

	h := mr.Header
	setEnvelope(&email, h)

	if !checkEnvelopeCondition(email, search) {
		if err := fetchCmd.Close(); err != nil {
			errChan <- err
			return
		}
		errChan <- nil
		ch <- Email{}
		return
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
			errChan <- err
		} else {
			errChan <- nil
			ch <- Email{}
		}
	} else {
		errChan <- fetchCmd.Close()
		ch <- email
	}
}

func fetchEmails(c *imapclient.Client, search Search) ([]Email, error) {

	selectedMbox, err := c.Select(search.Mailbox, nil).Wait()
	if err != nil {
		return nil, err
	}

	if selectedMbox.NumMessages == 0 {
		return []Email{}, nil
	}

	emails := make([]Email, cmp.Or(search.Limit, EmailReadingDefaultCapacity))
	ch := make(chan Email)
	errChan := make(chan error)
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{{}},
	}

	for i := 0; i < len(emails); i++ {
		var seqSet imap.SeqSet
		seqSet.AddNum(uint32(len(emails) - i))
		go fetchEmail(c, search, seqSet, *fetchOptions, ch, errChan)
	}
        index := 0
	for i := 0; i < len(emails); i++ {
		if err := <-errChan; err != nil {
			return nil, err
		}
		emails[index] = <-ch
		if (!reflect.DeepEqual(emails[index], Email{})) {
			index++ // ignore empty emails
		}
	}

	sort.Slice(emails[:index], func(i, j int) bool {return emails[i].Date > emails[j].Date})
	
	return emails[:index], nil
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
		if !(email.From == search.SearchFrom) {
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
