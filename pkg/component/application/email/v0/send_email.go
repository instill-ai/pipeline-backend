package email

import (
	"fmt"
	"net/smtp"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type SendEmailInput struct {
	Recipients []string `json:"recipients"`
	Cc         []string `json:"cc,omitempty"`
	Bcc        []string `json:"bcc,omitempty"`
	Subject    string   `json:"subject"`
	Message    string   `json:"message"`
}

type SendEmailOutput struct {
	Result string `json:"result"`
}

func (e *execution) sendEmail(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := SendEmailInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, err
	}

	setup := e.GetSetup()
	smtpHost := setup.GetFields()["server-address"].GetStringValue()
	smtpPort := setup.GetFields()["server-port"].GetNumberValue()
	from := setup.GetFields()["email-address"].GetStringValue()
	password := setup.GetFields()["password"].GetStringValue()

	auth := smtp.PlainAuth(
		"",
		from,
		password,
		smtpHost,
	)

	recipients := inputStruct.Recipients
	bcc := inputStruct.Bcc
	message := buildMessage(from, recipients, inputStruct.Cc, inputStruct.Subject, inputStruct.Message)

	// Fix bug 503 5.5.1 RCPT first
	if len(bcc) == 0 {
		bcc = append(bcc, from)
	}

	err = smtp.SendMail(fmt.Sprintf("%v:%v", smtpHost, smtpPort), auth, from, bcc, []byte(message))
	if err != nil {
		return nil, fmt.Errorf("failed to send email: %v", err)
	}

	outputStruct := SendEmailOutput{
		Result: "Email sent successfully",
	}

	return base.ConvertToStructpb(outputStruct)
}

func buildMessage(from string, to []string, cc []string, subject string, body string) string {
	message := "From: " + from + "\n"

	if len(to) > 0 {
		message += "To: "
	}

	for i, recipient := range to {
		message += recipient
		if i != len(to)-1 {
			message += ","
		}
	}

	if len(cc) > 0 {
		message += "\nCc: "
	}

	for i, ccRecipient := range cc {
		message += ccRecipient

		if i != len(cc)-1 {
			message += ","
		}
	}
	if subject != "" {
		message += "\nSubject: " + subject
	}
	message += "\n\n" + body

	return message

}
