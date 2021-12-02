package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/serenize/snaker"
	log "github.com/sirupsen/logrus"
)

type collector struct {
	client influx.Client
	query  influx.Query
}

func newCollector(config influx.HTTPConfig) collector {
	log.Infof("Using InfluxDB at %v", *influxUrl)
	client, err := influx.NewHTTPClient(config)
	if err != nil {
		log.WithError(err).Panic("Failed to set up influx client")
	}

	return collector{
		client: client,
		query:  influx.NewQuery("SHOW STATS", "", ""),
	}
}

func (c collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("influxdb_exporter", "Bogus desc", []string{}, prometheus.Labels{})
}

func (c collector) Collect(ch chan<- prometheus.Metric) {
	t := time.Now()
	r, err := c.client.Query(c.query)
	ch <- prometheus.MustNewConstMetric(queryDuration, prometheus.GaugeValue, time.Since(t).Seconds())

	if err != nil {
		log.WithError(err).Error("SHOW STATS query failed")
		ch <- prometheus.MustNewConstMetric(querySuccess, prometheus.GaugeValue, 0)
		return
	} else if r.Error() != nil {
		log.WithError(r.Error()).Error("SHOW STATS query failed")
		ch <- prometheus.MustNewConstMetric(querySuccess, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(querySuccess, prometheus.GaugeValue, 1)

	uuids := make(map[string]struct{})
	for _, res := range r.Results {
		for _, s := range res.Series {
			for idx := 0; idx < len(s.Columns); idx++ {
				seriesName := strings.ToLower(snaker.CamelToSnake(s.Name))
				colName := strings.ToLower(snaker.CamelToSnake(s.Columns[idx]))
				fqName := fmt.Sprintf("influxdb_%s_%s", seriesName, colName)

				uuid := fmt.Sprintf("%v_%v_%v", seriesName, colName, s.Tags)
				if _, exist := uuids[uuid]; exist {
					log.Errorf("repeated uuid: %v", uuid)
					continue
				} else {
					uuids[uuid] = struct{}{}
				}

				desc := prometheus.NewDesc(fqName, colName, []string{}, s.Tags)

				asNum, ok := s.Values[0][idx].(json.Number)
				if !ok {
					log.
						WithFields(log.Fields{"series": s.Name, "column": colName, "value": s.Values[0][idx]}).
						Warn("Failed to convert value to number")
				}
				val, err := asNum.Float64()
				if err != nil {
					log.WithFields(log.Fields{"series": s.Name, "column": colName, "value": s.Values[0][idx]}).
						Warn("Failed to convert value to float")
				} else {
					m := prometheus.MustNewConstMetric(desc, prometheus.UntypedValue, val)
					ch <- m
				}
			}
		}
	}
}
