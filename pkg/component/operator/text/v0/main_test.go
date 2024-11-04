// TestCleanData tests the CleanData function
func TestCleanData(t *testing.T) {
	c := quicktest.New(t)

	testCases := []TestCase{
		{
			name: "Valid Input",
			input: &CleanDataInput{
				Texts: []string{"Sample text 1.", "Sample text 2."},
				Setting: DataCleaningSetting{ // Make sure this matches the struct definition
					CleanMethod:    "Regex",
					ExcludePatterns: []string{"exclude this"}, // Ensure correct type
				},
			},
			want: &CleanDataOutput{
				CleanedTexts: []string{"Sample text 1.", "Sample text 2."}, // Expected cleaned output
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			output := CleanData(*tc.input) // Use dereference for CleanData
			c.Assert(output, quicktest.DeepEquals, tc.want)
		})
	}
}

// TestRegexFunctionality tests the regex cleaning functions
func TestRegexFunctionality(t *testing.T) {
	c := quicktest.New(t)

	c.Run("Clean Text Using Regex", func(c *quicktest.C) {
		input := []string{"Sample text with exclude this pattern."} // Change to []string
		expectedOutput := []string{"Sample text with  pattern."} // Expected output after cleaning

		output := cleanTextUsingRegex(input, []string{"exclude this"}) // Ensure the first argument is []string
		c.Assert(output, quicktest.DeepEquals, expectedOutput) // Match expected output type
	})

	c.Run("Clean Text Using Substring", func(c *quicktest.C) {
		input := []string{"Sample text without any exclusion."} // Change to []string
		expectedOutput := []string{"Sample text without any exclusion."}

		output := cleanTextUsingSubstring(input, "exclude") // Ensure correct parameters are passed
		c.Assert(output, quicktest.DeepEquals, expectedOutput)
	})
}

// TestCompileRegexPatterns tests the compileRegexPatterns function
func TestCompileRegexPatterns(t *testing.T) {
	c := quicktest.New(t)

	c.Run("Compile Patterns", func(c *quicktest.C) {
		patterns := []string{"exclude this"}
		compiled, err := compileRegexPatterns(patterns) // Ensure you're capturing all return values
		c.Assert(err, quicktest.IsNil) // Check for error
		c.Assert(len(compiled), quicktest.Equals, 1) // Expect one compiled pattern
	})
}
