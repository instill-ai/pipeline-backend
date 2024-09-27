package email

// IMAP client is not interface, so it only focuses on testing
// the specific logic of the email component.
import (
	"testing"

	"github.com/emersion/go-message/mail"
	"github.com/frankban/quicktest"
)

func TestSetEnvelope(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name           string
		inputHeaderMap map[string][]string
		expected       Email
	}{
		{
			name: "set envelope with all information",
			inputHeaderMap: map[string][]string{
				"Date":    {"01 Jan 2024 00:00:00 +0000"},
				"From":    {"fakeFrom@gmail.com"},
				"To":      {"fakeTo@gmail.com"},
				"Subject": {"fake subject"},
			},
			expected: Email{
				Date:    "2024-01-01 00:00:00",
				From:    "<fakeFrom@gmail.com>",
				Subject: "fake subject",
				To:      []string{"<fakeTo@gmail.com>"},
			},
		},
		{
			name: "set envelope with missing information",
			inputHeaderMap: map[string][]string{
				"Date":    {"01 Jan 2024 00:00:00 +0000"},
				"From":    {"fakeFrom@gmail.com"},
				"Subject": {"fake subject"},
			},
			expected: Email{
				Date:    "2024-01-01 00:00:00",
				From:    "<fakeFrom@gmail.com>",
				Subject: "fake subject",
				To:      []string{},
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			input := mail.HeaderFromMap(tc.inputHeaderMap)
			email := Email{}
			setEnvelope(&email, input)

			c.Assert(email.Date, quicktest.Equals, tc.expected.Date)
			c.Assert(email.From, quicktest.Equals, tc.expected.From)
			c.Assert(email.Subject, quicktest.Equals, tc.expected.Subject)
			c.Assert(email.To, quicktest.ContentEquals, tc.expected.To)

		})
	}
}

func TestCheckEnvelopeCondition(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name     string
		email    Email
		search   Search
		expected bool
	}{
		{
			name: "include search condition with all information",
			email: Email{
				Date:    "2024-01-01 00:00:00",
				From:    "<fakeFrom@gmail.com>",
				Subject: "fake subject",
				To:      []string{"fakeTo@gmail.com", "fakeTo123@gmail.com"},
			},
			search: Search{
				SearchSubject: "fake",
				SearchFrom:    "<fakeFrom@gmail.com>",
				SearchTo:      "fakeTo@gmail.com",
				Date:          "2024-01-01",
			},
			expected: true,
		},
		{
			name: "include search condition: subject NOT match",
			email: Email{
				Date:    "2024-01-01 00:00:00",
				From:    "<fakeFrom@gmail.com>",
				Subject: "fake subjec",
				To:      []string{"fakeTo@gmail.com"},
			},
			search: Search{
				SearchSubject: "fake 1",
				SearchFrom:    "fakeFrom",
				SearchTo:      "fakeTo",
				Date:          "2024-01-01",
			},
			expected: false,
		},
		{
			name: "include search condition: search from email NOT match",
			email: Email{
				Date:    "2024-01-01 00:00:00",
				From:    "<fakeFrom@gmail.com>",
				Subject: "fake subject",
				To:      []string{"fakeTo@gmail.com"},
			},
			search: Search{
				SearchSubject: "fake",
				SearchFrom:    "fakeFrom1",
				SearchTo:      "fakeTo",
				Date:          "2024-01-01",
			},
			expected: false,
		},
		{
			name: "include search condition: search to email NOT match",
			email: Email{
				Date:    "2024-01-01 00:00:00",
				From:    "<fakeFrom@gmail.com>",
				Subject: "fake subject",
				To:      []string{"fakeTo@gmail.com"},
			},
			search: Search{
				SearchSubject: "fake",
				SearchFrom:    "fakeFrom",
				SearchTo:      "fakeTo2",
				Date:          "2024-01-01",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			c.Assert(checkEnvelopeCondition(tc.email, tc.search), quicktest.Equals, tc.expected)
		})
	}
}

func TestCheckMessageCondition(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name     string
		email    Email
		search   Search
		expected bool
	}{
		{
			name: "include search message: match",
			email: Email{
				Message: "Hello world",
			},
			search: Search{
				SearchEmailMessage: "Hello",
			},
			expected: true,
		},
		{
			name: "include search message: NOT match",
			email: Email{
				Message: "Hello world",
			},
			search: Search{
				SearchEmailMessage: "Hello1",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			c.Assert(checkMessageCondition(tc.email, tc.search), quicktest.Equals, tc.expected)
		})
	}
}

func TestIsHTMLType(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "is HTML type",
			contentType: "text/html; charset=utf-8",
			expected:    true,
		},
		{
			name:        "is NOT HTML type",
			contentType: "text/plain; charset=utf-8",
			expected:    false,
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			var th mail.InlineHeader
			th.Set("Content-Type", tc.contentType)
			c.Assert(isHTMLType(&th), quicktest.Equals, tc.expected)
		})
	}

}
