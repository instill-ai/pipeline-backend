package email

import (
	"testing"

	"github.com/frankban/quicktest"
)

// SMTP is not interface, so it only focuses on testing
// the specific logic of the email component.
func TestBuildMessage(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name    string
		from    string
		to      []string
		cc      []string
		subject string
		body    string
		want    string
	}{
		{
			name: "build message with to and len(to) is 1",
			from: "fake@gmail.com",
			to: []string{
				"fakeTo1@gmail.com",
			},
			cc:      []string{},
			subject: "fake subject",
			body:    "fake body",
			want: `From: fake@gmail.com
To: fakeTo1@gmail.com
Subject: fake subject

fake body`,
		},
		{
			name: "build message with to and len(to) is 2",
			from: "fake@gmail.com",
			to: []string{
				"fakeTo1@gmail.com",
				"fakeTo2@gmail.com",
			},
			cc:      []string{},
			subject: "fake subject",
			body:    "fake body",
			want: `From: fake@gmail.com
To: fakeTo1@gmail.com,fakeTo2@gmail.com
Subject: fake subject

fake body`,
		},
		{
			name:    "build message without to",
			from:    "fake@gmail.com",
			to:      []string{},
			cc:      []string{},
			subject: "fake subject",
			body:    "fake body",
			want: `From: fake@gmail.com

Subject: fake subject

fake body`,
		},
		{
			name: "build message with cc and len(cc) is 1",
			from: "fake@gmail.com",
			to:   []string{},
			cc: []string{
				"fakeCc@gmail.com",
			},
			subject: "fake subject",
			body:    "fake body",
			want: `From: fake@gmail.com

Cc: fakeCc@gmail.com
Subject: fake subject

fake body`,
		},
		{
			name: "build message with cc and len(cc) is 2",
			from: "fake@gmail.com",
			to:   []string{},
			cc: []string{
				"fakeCc@gmail.com",
				"fakeCc2@gmail.com",
			},
			subject: "fake subject",
			body:    "fake body",
			want: `From: fake@gmail.com

Cc: fakeCc@gmail.com,fakeCc2@gmail.com
Subject: fake subject

fake body`,
		},
		{
			name:    "build message without subject",
			from:    "fake@gmail.com",
			to:      []string{},
			cc:      []string{},
			subject: "",
			body:    "fake body",
			want: `From: fake@gmail.com


fake body`,
		},
		{
			name:    "build message without body",
			from:    "fake@gmail.com",
			to:      []string{},
			cc:      []string{},
			subject: "fake subject",
			body:    "",
			want: `From: fake@gmail.com

Subject: fake subject

`,
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			got := buildMessage(tc.from, tc.to, tc.cc, tc.subject, tc.body)
			c.Assert(got, quicktest.Equals, tc.want)
		})
	}
}
