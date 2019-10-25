package trace

import (
	"github.com/opentracing/opentracing-go"
	zipkinot "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	reporterhttp "github.com/openzipkin/zipkin-go/reporter/http"
)


func NewZipkinTracer(reporterUrl string, hostname string, servicePort uint16) error {
	reporter := reporterhttp.NewReporter(reporterUrl)
	var localEndpoint = &model.Endpoint{ServiceName: hostname, Port: servicePort}
	sampler, err := zipkin.NewCountingSampler(1)
	if err != nil {
		return err
	}

	nativeTracer, err := zipkin.NewTracer(reporter, zipkin.WithSampler(sampler), zipkin.WithLocalEndpoint(localEndpoint))
	if err != nil {
	    return err
	}

	tracer := zipkinot.Wrap(nativeTracer)
	opentracing.SetGlobalTracer(tracer)

	return nil
}

