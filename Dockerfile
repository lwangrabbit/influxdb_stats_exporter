FROM alpine:3.15.0

EXPOSE 9424

WORKDIR ./

COPY influxdb_stats_exporter /influxdb_stats_exporter
RUN chmod +x /influxdb_stats_exporter

ENTRYPOINT ["/influxdb_stats_exporter"]

CMD ["--influx.url=http://localhost:8086", "--influx.user=", "--influx.password="]

