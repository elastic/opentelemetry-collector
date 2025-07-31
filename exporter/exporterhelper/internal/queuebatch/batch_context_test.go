// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestBatchContextLink(t *testing.T) {
	tracerProvider := componenttest.NewTelemetry().NewTelemetrySettings().TracerProvider
	tracer := tracerProvider.Tracer("go.opentelemetry.io/collector/exporter/exporterhelper")

	ctx1 := context.Background()

	ctx2, span2 := tracer.Start(ctx1, "span2")
	defer span2.End()

	ctx3, span3 := tracer.Start(ctx1, "span3")
	defer span3.End()

	ctx4, span4 := tracer.Start(ctx1, "span4")
	defer span4.End()

	batchContext := contextWithMergedLinks(ctx2, ctx3)
	batchContext = contextWithMergedLinks(batchContext, ctx4)

	actualLinks := LinksFromContext(batchContext)
	require.Len(t, actualLinks, 3)
	require.Equal(t, trace.SpanContextFromContext(ctx2), actualLinks[0].SpanContext)
	require.Equal(t, trace.SpanContextFromContext(ctx3), actualLinks[1].SpanContext)
	require.Equal(t, trace.SpanContextFromContext(ctx4), actualLinks[2].SpanContext)
}

func TestBatchContextLinkMetadataPropogation(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		metadata1, metadata2 map[string][]string
		panicMsg             string
		expectedMetadata     map[string][]string
	}{
		{
			name: "no_allowed_keys",
		},
		{
			name: "metadata1_allowed_keys",
			metadata1: map[string][]string{
				"x-elastic-project-id": []string{"pid1"},
			},
			panicMsg: "unexpected metadata keys, the partition has allowed metadata keys with different values",
		},
		{
			name: "metadata2_allowed_keys",
			metadata2: map[string][]string{
				"x-elastic-project-id": []string{"pid2"},
			},
			panicMsg: "unexpected metadata keys, the partition has allowed metadata keys with different values",
		},
		{
			name: "metadata_unequal",
			metadata1: map[string][]string{
				"x-elastic-project-id": []string{"pid1"},
			},
			metadata2: map[string][]string{
				"x-elastic-project-id": []string{"pid2"},
			},
			panicMsg: "unexpected metadata keys, the partition has allowed metadata keys with different values",
		},
		{
			name: "metadata_correct",
			metadata1: map[string][]string{
				"x-elastic-project-id": []string{"pid1"},
				"other-metadata-1":     []string{"other1"},
				"other-metadata-2":     []string{"other2"},
			},
			metadata2: map[string][]string{
				"x-elastic-project-id": []string{"pid1"},
				"other-metadata-1":     []string{"other1"},
				"other-metadata-3":     []string{"other3"},
			},
			expectedMetadata: map[string][]string{
				"x-elastic-project-id": []string{"pid1"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx1 := client.NewContext(
				context.Background(),
				client.Info{Metadata: client.NewMetadata(tc.metadata1)},
			)
			ctx2 := client.NewContext(
				context.Background(),
				client.Info{Metadata: client.NewMetadata(tc.metadata2)},
			)

			if tc.panicMsg == "" {
				require.NotPanics(t, func() { contextWithMergedLinks(ctx1, ctx2) })
				ctx := contextWithMergedLinks(ctx1, ctx2)
				actualMetadata := client.FromContext(ctx).Metadata
				for k := range actualMetadata.Keys() {
					assert.Equal(t, tc.expectedMetadata[k], actualMetadata.Get(k))
				}
			} else {
				require.PanicsWithValue(t, tc.panicMsg, func() { contextWithMergedLinks(ctx1, ctx2) })
			}
		})
	}
}
