package text

import (
	"os"
	"testing"

	"github.com/frankban/quicktest"
)

func Test_MarkdownSplitter(t *testing.T) {

	c := quicktest.New(t)

	testCases := []struct {
		input     ChunkTextInput
		outputLen int
	}{
		{
			input: ChunkTextInput{
				Text: `# asf65463
	## 654654
	fasdflj`,
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod:  "Markdown",
						ChunkSize:    800,
						ChunkOverlap: 200,
					},
				},
			},
			outputLen: 1,
		},
		{
			input: ChunkTextInput{
				Text: `# 醫囑
檢驗 : Urine routine(急) [尿液] [有蓋定量離心管(尿液收集管)] STAT 【註:Foley】
=> **尿液檢查採樣來源為Foley，表示有裝置導尿管**

# 護理
病人2way尿管存，管路引流順暢，尿液呈淡黃色，管路固定於右大腿，無滑脫，續觀
=> **尿管存，表示有裝置導尿管**

# 個案泌尿道感染判定
1. 有導尿管
2. 病患無UTI感染症狀
=> **判斷為:非泌尿道感染，僅無症狀菌尿症(not UTI; asymptomatic bacteuria only)**`,
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod:  "Markdown",
						ChunkSize:    800,
						ChunkOverlap: 200,
					},
				},
			},
			outputLen: 3,
		},
	}
	for _, testCase := range testCases {
		c.Run("Test bug cases reported", func(c *quicktest.C) {
			inputStruct := testCase.input
			setting := inputStruct.Strategy.Setting
			split := NewMarkdownTextSplitter(
				setting.ChunkSize,
				setting.ChunkOverlap,
				inputStruct.Text,
			)

			err := split.Validate()
			c.Assert(err, quicktest.IsNil)

			chunks, err := split.SplitText()

			c.Assert(err, quicktest.IsNil)

			c.Assert(len(chunks), quicktest.DeepEquals, testCase.outputLen)
		})
	}
}

func Test_PositionChecker(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		input        ChunkTextInput
		fileName     string
		outputChunks []ContentChunk
	}{
		{
			input: ChunkTextInput{
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod:  "Markdown",
						ChunkSize:    800,
						ChunkOverlap: 200,
					},
				},
			},
			fileName: "testdata/list_position_test.md",
			outputChunks: []ContentChunk{
				{
					Chunk:                "# Progress Notes\n\nLUTS  \n=> **LUTS:Lower Urinary Tract Symptoms，表示有下泌尿道感染**",
					ContentStartPosition: 17,
					ContentEndPosition:   73,
				},
				{
					Chunk:                "# 護理\n\n主訴有困難解尿情形，有尿液感但都解不出來   \n=> **符合有症狀的泌尿道感染之徵象或症狀:解尿困難或疼痛(dysuria)**",
					ContentStartPosition: 82,
					ContentEndPosition:   146,
				},
				{
					Chunk:                "# 個案泌尿道感染判定\n\n\n1. 沒有導尿管或導尿管留置未超過2天\n2. 病患有UTI感染症狀:解尿困難或疼痛(dysuria)、Lower Urinary Tract Symptoms  \n=> **判斷為:非導尿管相關泌尿道感染(Non-CAUTI)**\n\n",
					ContentStartPosition: 160,
					ContentEndPosition:   275,
				},
			},
		},
	}

	for _, testCase := range testCases {

		c.Run("Test bug cases reported about position", func(c *quicktest.C) {
			inputStruct := testCase.input

			bytes, err := os.ReadFile(testCase.fileName)
			c.Assert(err, quicktest.IsNil)

			inputStruct.Text = string(bytes)

			setting := inputStruct.Strategy.Setting
			split := NewMarkdownTextSplitter(
				setting.ChunkSize,
				setting.ChunkOverlap,
				inputStruct.Text,
			)

			err = split.Validate()
			c.Assert(err, quicktest.IsNil)

			chunks, err := split.SplitText()

			c.Assert(err, quicktest.IsNil)

			for i, chunk := range chunks {
				c.Assert(chunk.Chunk, quicktest.Equals, testCase.outputChunks[i].Chunk)
				c.Assert(chunk.ContentStartPosition, quicktest.Equals, testCase.outputChunks[i].ContentStartPosition)
				c.Assert(chunk.ContentEndPosition, quicktest.Equals, testCase.outputChunks[i].ContentEndPosition)
			}
		})
	}
}
