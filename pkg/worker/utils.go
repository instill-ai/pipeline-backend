package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/instill-ai/pipeline-backend/config"
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
