package data

import (
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestNewNumberFromFloat(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"Zero", 0.0, 0.0},
		{"Positive integer", 42.0, 42.0},
		{"Negative integer", -42.0, -42.0},
		{"Positive decimal", 3.14159, 3.14159},
		{"Negative decimal", -3.14159, -3.14159},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			num := NewNumberFromFloat(tc.input)
			c.Assert(num, qt.Not(qt.IsNil))
			c.Assert(num.Float64(), qt.Equals, tc.expected)
			c.Assert(num.IsInteger, qt.IsFalse)
		})
	}
}

func TestNewNumberFromInteger(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    int
		expected int
	}{
		{"Zero", 0, 0},
		{"Positive integer", 42, 42},
		{"Negative integer", -42, -42},
		{"Large positive", 999999, 999999},
		{"Large negative", -999999, -999999},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			num := NewNumberFromInteger(tc.input)
			c.Assert(num, qt.Not(qt.IsNil))
			c.Assert(num.Integer(), qt.Equals, tc.expected)
			c.Assert(num.IsInteger, qt.IsTrue)
		})
	}
}

func TestNumberConversion(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	c.Run("Float to Integer conversion", func(c *qt.C) {
		num := NewNumberFromFloat(42.9)
		c.Assert(num.Integer(), qt.Equals, 42)
	})

	c.Run("Integer to Float conversion", func(c *qt.C) {
		num := NewNumberFromInteger(42)
		c.Assert(num.Float64(), qt.Equals, 42.0)
	})
}

func TestNumberString(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name     string
		number   *numberData
		expected string
	}{
		{"Integer zero", NewNumberFromInteger(0), "0"},
		{"Float zero", NewNumberFromFloat(0.0), "0.000000"},
		{"Positive integer", NewNumberFromInteger(42), "42"},
		{"Negative integer", NewNumberFromInteger(-42), "-42"},
		{"Positive float", NewNumberFromFloat(3.14159), "3.141590"},
		{"Negative float", NewNumberFromFloat(-3.14159), "-3.141590"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			c.Assert(tc.number.String(), qt.Equals, tc.expected)
		})
	}
}

func TestNumberEqual(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name     string
		a        *numberData
		b        format.Value
		expected bool
	}{
		{"Equal integers", NewNumberFromInteger(42), NewNumberFromInteger(42), true},
		{"Equal floats", NewNumberFromFloat(3.14), NewNumberFromFloat(3.14), true},
		{"Integer equals float", NewNumberFromInteger(42), NewNumberFromFloat(42.0), true},
		{"Different numbers", NewNumberFromInteger(42), NewNumberFromInteger(43), false},
		{"Number vs null", NewNumberFromInteger(42), NewNull(), false},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			c.Assert(tc.a.Equal(tc.b), qt.Equals, tc.expected)
		})
	}
}
