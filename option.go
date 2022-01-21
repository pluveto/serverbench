package main

type EndPointType string

const (
	EndPoint_HTTP EndPointType = "http"
)

type Option struct {
	MaxCPUNum    int
	WorkerNum    int64
	BatchSize    int64
	EndPoint     string
	EndPointType EndPointType
	HttpMethod   string
}

type ArgProvider interface {
	GetHeaders(*RequestContext) map[string]string
	GetBody(*RequestContext) []byte
}
