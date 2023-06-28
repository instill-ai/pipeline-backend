package service

import (
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

func (s *service) WriteNewDataPoint(p *write.Point) {
	s.influxDBWriteClient.WritePoint(p)
}
