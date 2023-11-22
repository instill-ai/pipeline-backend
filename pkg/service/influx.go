package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"google.golang.org/protobuf/types/known/structpb"
)

// TODO: better naming for this file

func (s *service) WriteNewPipelineDataPoint(ctx context.Context, data utils.PipelineUsageMetricData) error {

	if config.Config.Server.Usage.Enabled {

		bData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", data.OwnerUID), string(bData))
	}

	s.influxDBWriteClient.WritePoint(utils.NewPipelineDataPoint(data))

	return nil
}

func (s *service) WriteNewConnectorDataPoint(ctx context.Context, data utils.ConnectorUsageMetricData, pipelineMetadata *structpb.Value) error {

	if config.Config.Server.Usage.Enabled {

		bData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:connector.execute_data", data.OwnerUID), string(bData))
	}

	s.influxDBWriteClient.WritePoint(utils.NewConnectorDataPoint(data, pipelineMetadata))

	return nil
}
