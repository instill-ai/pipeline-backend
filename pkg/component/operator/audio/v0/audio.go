package audio

type Audio string

const (
	sampleRate = 16000
	numChannel = 1
)

type segment struct {
	StartTime float64 `json:"start-time"`
	EndTime   float64 `json:"end-time"`
}
