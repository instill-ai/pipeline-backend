package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/data"
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

// setIteratorIndex converts the iterator index identifier into a numeric
// index. For example, it converts `${variable.array[i]}` into
// `${variable.array[0]}`.
func setIteratorIndex(v data.Value, identifier string, index int) data.Value {
	if identifier == "" {
		identifier = "i"
	}
	switch v := v.(type) {
	case *data.String:
		s := v.GetString()
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
	case *data.Array:
		m := data.NewArray(make([]data.Value, len(v.Values)))
		for idx, item := range v.Values {
			m.Values[idx] = setIteratorIndex(item, identifier, index)
		}
		return m
	case *data.Map:
		m := data.NewMap(make(map[string]data.Value))
		for k, v := range v.Fields {
			m.Fields[k] = setIteratorIndex(v, identifier, index)
		}
		return m
	default:
		return v
	}
}
