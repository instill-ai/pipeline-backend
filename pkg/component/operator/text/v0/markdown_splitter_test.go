package text

import (
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
