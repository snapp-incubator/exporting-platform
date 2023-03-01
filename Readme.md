# Exporting Platform

## Develop

To run the code in your development environment:

```shell
go run cmd/main.go
```

## Add Exporters

To add a new exporter, you should create a file inside [internal/exporters](internal/exporters) and add your new collector struct there and add new fields with type `*prometheus.Desc`. After that, you should create the New function for that struct and create `prometheus.Desc` objects there.

After creating New function, you should implement two methods for your struct.

- Describe: which passes a read-only channel of type `*prometheus.Desc` and you should pass each Desc object you had in struct to it.
- Collect: which passes a read-only channel of type `prometheus.Metric` and you should implement the exporting logic here. After that, you should return objects of type `prometheus.MustNewConstMetric` to the channel. 



