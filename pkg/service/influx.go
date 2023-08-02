package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
)

// TODO: better naming for this file

func (s *service) WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData) error {

	if config.Config.Server.Usage.Enabled {

		bData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", data.OwnerUID), string(bData))
	}

	s.influxDBWriteClient.WritePoint(utils.NewDataPoint(data))

	return nil
}
