// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgininter provides common utility functions for middleware of gin framework
package rkgininter

import (
	"github.com/gin-gonic/gin"
	"github.com/rookie-ninja/rk-common/common"
	"go.uber.org/zap"
	"net"
	"strings"
)

var (
	// Realm environment variable
	Realm = zap.String("realm", rkcommon.GetEnvValueOrDefault("REALM", "*"))
	// Region environment variable
	Region = zap.String("region", rkcommon.GetEnvValueOrDefault("REGION", "*"))
	// AZ environment variable
	AZ = zap.String("az", rkcommon.GetEnvValueOrDefault("AZ", "*"))
	// Domain environment variable
	Domain = zap.String("domain", rkcommon.GetEnvValueOrDefault("DOMAIN", "*"))
	// LocalIp read local IP from localhost
	LocalIp = zap.String("localIp", rkcommon.GetLocalIP())
	// LocalHostname read hostname from localhost
	LocalHostname = zap.String("localHostname", rkcommon.GetLocalHostname())
)

const (
	// RpcEntryNameKey entry name key
	RpcEntryNameKey = "ginEntryName"
	// RpcEntryNameValue entry name
	RpcEntryNameValue = "gin"
	// RpcEntryTypeValue entry type
	RpcEntryTypeValue = "gin"
	// RpcEventKey event key
	RpcEventKey = "ginEvent"
	// RpcLoggerKey logger key
	RpcLoggerKey = "ginLogger"
	// RpcTracerKey tracer key
	RpcTracerKey = "ginTracer"
	// RpcSpanKey span key
	RpcSpanKey = "ginSpan"
	// RpcTracerProviderKey trace provider key
	RpcTracerProviderKey = "ginTracerProvider"
	// RpcPropagatorKey propagator key
	RpcPropagatorKey = "ginPropagator"
	// RpcAuthorizationHeaderKey auth key
	RpcAuthorizationHeaderKey = "authorization"
	// RpcApiKeyHeaderKey api auth key
	RpcApiKeyHeaderKey = "X-API-Key"
	// RpcJwtTokenKey key of jwt token in context
	RpcJwtTokenKey = "ginJwt"
	// RpcCsrfTokenKey key of csrf token injected by csrf middleware
	RpcCsrfTokenKey = "ginCsrfToken"
)

// GetRemoteAddressSet returns remote endpoint information set including IP, Port.
// We will do as best as we can to determine it.
// If fails, then just return default ones.
func GetRemoteAddressSet(ctx *gin.Context) (remoteIp, remotePort string) {
	remoteIp, remotePort = "0.0.0.0", "0"

	if ctx == nil || ctx.Request == nil {
		return
	}

	var err error
	if remoteIp, remotePort, err = net.SplitHostPort(ctx.Request.RemoteAddr); err != nil {
		return
	}

	forwardedRemoteIp := ctx.GetHeader("x-forwarded-for")

	// Deal with forwarded remote ip
	if len(forwardedRemoteIp) > 0 {
		if forwardedRemoteIp == "::1" {
			forwardedRemoteIp = "localhost"
		}

		remoteIp = forwardedRemoteIp
	}

	if remoteIp == "::1" {
		remoteIp = "localhost"
	}

	return remoteIp, remotePort
}

// ShouldLog determines whether should log the RPC
func ShouldLog(ctx *gin.Context) bool {
	if ctx == nil || ctx.Request == nil {
		return false
	}

	// ignoring /rk/v1/assets, /rk/v1/tv and /sw/ path while logging since these are internal APIs.
	if strings.HasPrefix(ctx.Request.URL.Path, "/rk/v1/assets") ||
		strings.HasPrefix(ctx.Request.URL.Path, "/rk/v1/tv") ||
		strings.HasPrefix(ctx.Request.URL.Path, "/sw/") {
		return false
	}

	return true
}
