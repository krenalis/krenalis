# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

"""Generate the driver-only fault hooks used by the tagged chunk test."""

from pathlib import Path


DRIVER = "vendor/github.com/snowflakedb/gosnowflake/v2/"
SUPPORT = '''//go:build krenalis_chunkfault

package gosnowflake

import (
    "context"
    "errors"
    "sync"
    "sync/atomic"
)

// ErrChunkFault identifies the injected, retryable download failure.
var ErrChunkFault = errors.New("Snowflake test: injected chunk download failure")

// ChunkFaultObservation counts delivery and failures for one marked request.
type ChunkFaultObservation struct {
    Rows atomic.Int64
    Chunks atomic.Int64
    Attempts atomic.Int64
    firstRow chan struct{}
    first sync.Once
}

// WithChunkFault fails the second remote chunk after the first delivered row.
func WithChunkFault(ctx context.Context) (context.Context, *ChunkFaultObservation) {
    observation := &ChunkFaultObservation{firstRow: make(chan struct{})}
    return context.WithValue(ctx, chunkFaultKey{}, observation), observation
}

// ExpectedAttempts is the initial attempt plus every downloader retry.
func (observation *ChunkFaultObservation) ExpectedAttempts() int64 {
    return int64(maxChunkDownloaderErrorCounter) + 1
}

type chunkFaultKey struct{}

func chunkFaultObservation(ctx context.Context) *ChunkFaultObservation {
    tag, _ := ctx.Value(queryTag).(string)
    if tag != "krenalis-transport/record-query" {
        return nil
    }
    observation, _ := ctx.Value(chunkFaultKey{}).(*ChunkFaultObservation)
    return observation
}
'''


def prepare_chunk_fault(root, directory, replacements):
    edits = {
        "connection_util.go": [(
            "\treturn &snowflakeChunkDownloader{",
            "\tif observation := chunkFaultObservation(ctx); observation != nil {\n"
            "\t\tobservation.Chunks.Store(int64(len(data.Chunks)))\n\t}\n"
            "\treturn &snowflakeChunkDownloader{",
        )],
        "chunk_downloader.go": [(
            "func downloadChunkHelper(ctx context.Context, scd *snowflakeChunkDownloader, idx int) error {\n",
            "func downloadChunkHelper(ctx context.Context, scd *snowflakeChunkDownloader, idx int) error {\n"
            "\tif observation := chunkFaultObservation(ctx); observation != nil && idx == 1 {\n"
            "\t\tselect {\n\t\tcase <-observation.firstRow:\n"
            "\t\tcase <-ctx.Done():\n\t\t\treturn ctx.Err()\n\t\t}\n"
            "\t\tobservation.Attempts.Add(1)\n\t\treturn ErrChunkFault\n\t}\n",
        ), (
            '\t\tlogger.WithContext(ctx).Errorf(\n'
            '\t\t\t"failed to extract HTTP response body. URL: %v, err: %v", scd.ChunkMetas[idx].URL, err)',
            '\t\tif chunkFaultObservation(ctx) != nil {\n'
            '\t\t\tlogger.WithContext(ctx).Errorf("Snowflake test chunk failure index=%d", idx)\n'
            '\t\t} else {\n'
            '\t\t\tlogger.WithContext(ctx).Errorf(\n'
            '\t\t\t\t"failed to extract HTTP response body. URL: %v, err: %v", scd.ChunkMetas[idx].URL, err)\n'
            '\t\t}',
        )],
        "rows.go": [(
            '\treturn err\n}\n\nfunc (rows *snowflakeRows) HasNextResultSet()',
            '\tif observation := chunkFaultObservation(rows.ctx); observation != nil {\n'
            '\t\tobservation.Rows.Add(1)\n'
            '\t\tobservation.first.Do(func() { close(observation.firstRow) })\n\t}\n'
            '\treturn err\n}\n\nfunc (rows *snowflakeRows) HasNextResultSet()',
        )],
    }
    for name, changes in edits.items():
        source = root / DRIVER / name
        text = replacements[str(source)][1] if str(source) in replacements else source.read_text()
        for old, new in changes:
            if text.count(old) != 1:
                raise ValueError("expected one chunk fault overlay anchor in " + str(Path(DRIVER) / name))
            text = text.replace(old, new)
        replacements[str(source)] = (directory / DRIVER / name, text)
    source = root / DRIVER / "chunk_fault.go"
    if source.exists():
        raise ValueError("chunk fault support must not replace an existing driver file")
    replacements[str(source)] = (directory / DRIVER / "chunk_fault.go", SUPPORT)
