package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/utils"
)

func (w *worker) writeNewDataPoint(ctx context.Context, data utils.UsageMetricData) {
	w.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.trigger_time", data.OwnerUID), data.TriggerTime.Format(time.RFC3339Nano))
	w.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.pipeline_uid", data.OwnerUID), data.PipelineUID)
	w.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.trigger_uid", data.OwnerUID), data.PipelineTriggerUID)
	w.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.trigger_mode", data.OwnerUID), data.TriggerMode.String())
	w.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.status", data.OwnerUID), data.Status.String())

	w.influxDBWriteClient.WritePoint(utils.NewDataPoint(data))
}
