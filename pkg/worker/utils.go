package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
)

func (w *worker) writeNewDataPoint(ctx context.Context, data utils.PipelineUsageMetricData) error {
	if config.Config.Server.Usage.Enabled {
		bData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		w.redisClient.RPush(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", data.OwnerUID), string(bData))
	}

	w.influxDBWriteClient.WritePoint(utils.NewPipelineDataPoint(data))
	w.influxDBWriteClient.WritePoint(utils.DeprecatedNewPipelineDatapoint(data))

	return nil
}

const (
	rangeStart             = "start"
	rangeStop              = "stop"
	rangeStep              = "step"
	defaultRangeIdentifier = "i"
)

// setIteratorIndex converts the iterator index identifier into a numeric
// index. For example, it converts `${variable.array[i]}` into
// `${variable.array[0]}`.
func setIteratorIndex(v format.Value, identifier string, index int) format.Value {
	if identifier == "" {
		identifier = defaultRangeIdentifier
	}
	switch v := v.(type) {
	case format.String:
		s := v.String()
		val := ""
		for {
			startIdx := strings.Index(s, "${")
			if startIdx == -1 {
				val += s
				break
			}
			val += s[:startIdx]
			s = s[startIdx:]
			endIdx := strings.Index(s, "}")
			if endIdx == -1 {
				val += s
				break
			}

			ref := strings.TrimSpace(s[2:endIdx])
			ref = strings.ReplaceAll(ref, fmt.Sprintf("[%s]", identifier), fmt.Sprintf("[%d]", index))

			val += fmt.Sprintf("${%s}", ref)
			s = s[endIdx+1:]
		}
		return data.NewString(val)
	case data.Array:
		m := make(data.Array, len(v))
		for idx, item := range v {
			m[idx] = setIteratorIndex(item, identifier, index)
		}
		return m
	case data.Map:
		m := data.Map{}
		for k, v := range v {
			m[k] = setIteratorIndex(v, identifier, index)
		}
		return m
	default:
		return v
	}
}
