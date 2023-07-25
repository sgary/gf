// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package tracing provides some utility functions for tracing functionality.
package tracing

import (
	"context"
	"github.com/smallnest/rpcx/share"
	"go.opentelemetry.io/otel/propagation"
	"math"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/gogf/gf/v2/container/gtype"
	"github.com/gogf/gf/v2/encoding/gbinary"
	"github.com/gogf/gf/v2/util/grand"
)

var (
	randomInitSequence = int32(grand.Intn(math.MaxInt32))
	sequence           = gtype.NewInt32(randomInitSequence)
)

// NewIDs creates and returns a new trace and span ID.
func NewIDs() (traceID trace.TraceID, spanID trace.SpanID) {
	return NewTraceID(), NewSpanID()
}

// NewTraceID creates and returns a trace ID.
func NewTraceID() (traceID trace.TraceID) {
	var (
		timestampNanoBytes = gbinary.EncodeInt64(time.Now().UnixNano())
		sequenceBytes      = gbinary.EncodeInt32(sequence.Add(1))
		randomBytes        = grand.B(4)
	)
	copy(traceID[:], timestampNanoBytes)
	copy(traceID[8:], sequenceBytes)
	copy(traceID[12:], randomBytes)
	return
}

// NewSpanID creates and returns a span ID.
func NewSpanID() (spanID trace.SpanID) {
	copy(spanID[:], gbinary.EncodeInt64(time.Now().UnixNano()/1e3))
	copy(spanID[4:], grand.B(4))
	return
}

// 兼容RPCX Context
type metadataSupplier struct {
	metadata map[string]string
}

var _ propagation.TextMapCarrier = &metadataSupplier{}

func (s *metadataSupplier) Get(key string) string {
	return s.metadata[key]
}

func (s *metadataSupplier) Set(key string, value string) {
	s.metadata[key] = value
}

func (s *metadataSupplier) Keys() []string {
	out := make([]string, 0, len(s.metadata))
	for key := range s.metadata {
		out = append(out, key)
	}
	return out
}

func Inject(ctx context.Context, propagators propagation.TextMapPropagator) {
	meta := ctx.Value(share.ReqMetaDataKey)
	if meta == nil {
		meta = make(map[string]string)
		if rpcxContext, ok := ctx.(*share.Context); ok {
			rpcxContext.SetValue(share.ReqMetaDataKey, meta)
		}
	}

	propagators.Inject(ctx, &metadataSupplier{
		metadata: meta.(map[string]string),
	})
}

func Extract(ctx context.Context, propagators propagation.TextMapPropagator) trace.SpanContext {
	meta := ctx.Value(share.ReqMetaDataKey)
	if meta == nil {
		meta = make(map[string]string)
		if rpcxContext, ok := ctx.(*share.Context); ok {
			rpcxContext.SetValue(share.ReqMetaDataKey, meta)
		}
	}

	ctx = propagators.Extract(ctx, &metadataSupplier{
		metadata: meta.(map[string]string),
	})

	return trace.SpanContextFromContext(ctx)
}
