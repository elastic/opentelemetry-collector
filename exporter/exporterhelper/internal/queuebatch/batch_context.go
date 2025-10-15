// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"slices"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/otel/trace"
)

type traceContextKeyType int

const batchSpanLinksKey traceContextKeyType = iota

// allowedContextKeys is a hardcoded list of context keys that are added
// to the context propogated by the batcher. This is an interim solution
// until https://github.com/open-telemetry/opentelemetry-collector/issues/13320
// is resolved and is kept simple for better maintainability across updates.
var allowedContextKeys = []string{
	"x-elastic-project-id",
	"x-elastic-target-id",
	"x-elastic-target-type",
	"x-elastic-mapping-mode",
}

// LinksFromContext returns a list of trace links registered in the context.
func LinksFromContext(ctx context.Context) []trace.Link {
	if ctx == nil {
		return []trace.Link{}
	}
	if links, ok := ctx.Value(batchSpanLinksKey).([]trace.Link); ok {
		return links
	}
	return []trace.Link{}
}

func parentsFromContext(ctx context.Context) []trace.Link {
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		return []trace.Link{{SpanContext: spanCtx}}
	}
	return LinksFromContext(ctx)
}

func contextWithMergedLinks(mergedCtx, ctx1, ctx2 context.Context) context.Context {
	meta1 := client.FromContext(ctx1).Metadata
	meta2 := client.FromContext(ctx2).Metadata
	m := make(map[string][]string)
	for _, ak := range allowedContextKeys {
		v1 := meta1.Get(ak)
		v2 := meta2.Get(ak)
		if len(v1) == 0 && len(v2) == 0 {
			continue
		}
		if !slices.Equal(v1, v2) {
			panic("unexpected metadata keys, the partition has allowed metadata keys with different values")
		}
		m[ak] = slices.Clone(v1)
	}
	return context.WithValue(
		client.NewContext(mergedCtx, client.Info{Metadata: client.NewMetadata(m)}),
		batchSpanLinksKey,
		append(parentsFromContext(ctx1), parentsFromContext(ctx2)...))
}
