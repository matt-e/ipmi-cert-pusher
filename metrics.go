package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	checkRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ipmi_cert_pusher_check_requests_total",
		Help: "Total number of server checks performed.",
	}, []string{"server"})

	checkErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ipmi_cert_pusher_check_errors_total",
		Help: "Total number of errors during server checks.",
	}, []string{"server", "stage"})

	checkDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ipmi_cert_pusher_check_duration_seconds",
		Help:    "Duration of each checkServer call in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"server"})

	pushTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ipmi_cert_pusher_push_total",
		Help: "Total number of successful SAA certificate pushes.",
	}, []string{"server"})

	certificateExpirySeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ipmi_cert_pusher_certificate_expiry_seconds",
		Help: "Seconds until the local certificate expires.",
	}, []string{"server"})
)
