package service

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/utils"
)

// TODO: better naming for this file

func (s *service) WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData) {

	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.trigger_time", data.OwnerUID), data.TriggerTime.Format(time.RFC3339Nano))
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.pipeline_uid", data.OwnerUID), data.PipelineUID)
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.trigger_uid", data.OwnerUID), data.PipelineTriggerUID)
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.trigger_mode", data.OwnerUID), data.TriggerMode.String())
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.status", data.OwnerUID), data.Status.String())

	s.influxDBWriteClient.WritePoint(utils.NewDataPoint(data))
}
