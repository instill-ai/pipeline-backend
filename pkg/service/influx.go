package service

import (
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	"github.com/instill-ai/pipeline-backend/config"
)

func (s *service) WriteNewDataPoint(p *write.Point) {
	// TODO: influxDB will be updated to requied instead of optional
	// will not need to consider the case if influxdb is not present
	if config.Config.Log.External {
		s.influxDBWriteClient.WritePoint(p)
	}
}
