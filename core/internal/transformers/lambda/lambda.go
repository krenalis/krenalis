//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package lambda

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/meergo/meergo/core/backoff"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	"github.com/meergo/meergo/core/internal/transformers/embed"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/metrics"
	"github.com/meergo/meergo/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go"
)

type function struct {
	settings Settings
	client   *lambda.Client
}

type Settings struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Role            string
	Node            struct {
		Runtime string
		Layer   string
	}
	Python struct {
		Runtime string
		Layer   string
	}
}

// New returns a new provider for Lambda with the given settings.
// Supports every Node and Python 3 runtime.
func New(settings Settings) transformers.FunctionProvider {
	return &function{settings: settings}
}

// Call calls the function with the given identifier and version for each record
// updating its Properties field with the result of each invocation.
//
// Before transformation, record properties must conform to inSchema.
// After transformation, they should conform to outSchema, unless an error
// occurs on the record.
//
// If the function does not exist, Call returns an ErrFunctionNotExist error.
// If the function exists but has an issue preventing execution (e.g., a syntax
// error), it returns a FunctionExecError.
// Even if the call succeeds, individual records may still encounter errors,
// which are stored in the Err field of each record.
func (fn *function) Call(ctx context.Context, id, version string, inSchema, outSchema types.Type, preserveJSON bool, records []transformers.Record) error {

	arn, language, err := parseID(id)
	if err != nil {
		return err
	}

	client, err := fn.connect(ctx)
	if err != nil {
		errorsMetric[errorTypeConnection].Inc()
		return err
	}

	// Marshal the values.
	payload := make([]byte, 0, 1024)
	payload = append(payload, '"')
	payload, err = transformers.Marshal(payload, inSchema, records, language, preserveJSON)
	if err != nil {
		errorsMetric[errorTypeSerialization].Inc()
		return err
	}
	payload = append(payload, '"')

	// Duration of a successful execution.
	var duration time.Duration

	// Invoke the function.
	var out *lambda.InvokeOutput
	bo := backoff.New(100)
	bo.SetCap(3 * time.Second)
	for bo.Next(ctx) {
		start := time.Now()
		out, err = client.Invoke(ctx, &lambda.InvokeInput{
			FunctionName: &arn,
			Payload:      payload,
			Qualifier:    &version,
		})
		if err != nil {
			if status, ok := httpStatusCode(err); ok {
				switch status {
				case 404:
					errorsMetric[errorTypeFunctionNotFound].Inc()
					return transformers.ErrFunctionNotExist
				case 409:
					// The function is pending.
					// Set the base with a greater value and retry.
					bo.SetBase(300)
					continue
				}
				if 500 <= status && status <= 599 {
					// There was an internal error.
					// Set the base with the default value and retry.
					errorsMetric[errorTypeLambdaInternal].Inc()
					bo.SetBase(100)
					continue
				}
			}
			errorsMetric[errorTypeNetwork].Inc()
			return err
		}
		duration = time.Since(start)
		break
	}
	if err = ctx.Err(); err != nil {
		return err
	}

	// Unmarshal the records.
	if out.FunctionError != nil {
		payload := struct {
			ErrorMessage string `json:"errorMessage"`
		}{}
		err := json.Unmarshal(out.Payload, &payload)
		if err != nil {
			errorsMetric[errorTypeLambdaInternal].Inc()
			return fmt.Errorf("transformers/lambda: cannot decode response executing function %q: %s", id, err)
		}
		return errors.New(payload.ErrorMessage)
	}
	var r io.Reader
	switch language {
	case state.JavaScript:
		r = bytes.NewReader(out.Payload)
	case state.Python:
		var s string
		err = json.Unmarshal(out.Payload, &s)
		if err != nil {
			errorsMetric[errorTypeSerialization].Inc()
			return fmt.Errorf("transformers/lambda: cannot decode response executing function %q: %s", id, err)
		}
		r = strings.NewReader(s)
	}

	// Unmarshal returns a FunctionExecError if execution fails, for example, due to a syntax error in the function.
	err = transformers.Unmarshal(r, records, outSchema, language, preserveJSON)
	if err != nil {
		if _, ok := err.(transformers.FunctionExecError); ok {
			errorsMetric[errorTypeFunctionExec].Inc()
		} else {
			errorsMetric[errorTypeSerialization].Inc()
		}
		return err
	}

	// Success metrics.
	durationMetric.Observe(duration.Seconds())
	recordsMetric.Add(len(records))

	return nil
}

// Close closes the function.
func (fn *function) Close(ctx context.Context) error {
	fn.client = nil
	return nil
}

// Create creates a new function with the given name, language, and source and
// returns its identifier and version.
func (fn *function) Create(ctx context.Context, name string, language state.Language, source string) (string, string, error) {
	if !transformers.ValidFunctionName(name) {
		return "", "", errors.New("function name is not valid")
	}
	if !fn.SupportLanguage(language) {
		return "", "", errors.New("language is not supported")
	}
	code, err := fn.code(source, language)
	if err != nil {
		return "", "", err
	}
	client, err := fn.connect(ctx)
	if err != nil {
		return "", "", err
	}
	var runtime string
	var layers []string
	switch language {
	case state.JavaScript:
		runtime = fn.settings.Node.Runtime
		if layer := fn.settings.Node.Layer; layer != "" {
			layers = []string{layer}
		}
	case state.Python:
		runtime = fn.settings.Python.Runtime
		if layer := fn.settings.Python.Layer; layer != "" {
			layers = []string{layer}
		}
	}
	out, err := client.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(name),
		Handler:      aws.String("index._handler"),
		Publish:      true,
		Role:         aws.String(fn.settings.Role),
		Runtime:      lambdatypes.Runtime(runtime),
		Code:         &lambdatypes.FunctionCode{ZipFile: code},
		Layers:       layers,
	})
	if err != nil {
		if status, ok := httpStatusCode(err); ok && status == 409 {
			return "", "", fmt.Errorf("transformers/lambda: function name %q already exists", name)
		}
		return "", "", err
	}
	if len(*out.FunctionArn) > 1000 {
		return "", "", fmt.Errorf("transformers/lambda: function ARN %q is too long", *out.FunctionArn)
	}
	if len(*out.Version) > 128 {
		return "", "", fmt.Errorf("transformers/lambda: version %q is too long", *out.Version)
	}
	var ext string
	switch language {
	case state.JavaScript:
		ext = "js"
	case state.Python:
		ext = "py"
	}
	id := *out.FunctionArn + "." + ext
	version := *out.Version
	return id, version, nil
}

// Delete deletes the function with the given identifier.
// If a function with the given identifier does not exist, it does nothing.
func (fn *function) Delete(ctx context.Context, id string) error {
	arn, _, err := parseID(id)
	if err != nil {
		return err
	}
	client, err := fn.connect(ctx)
	if err != nil {
		return err
	}
	_, err = client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: &arn,
	})
	if status, ok := httpStatusCode(err); ok && status == 404 {
		err = nil
	}
	return err
}

// SupportLanguage reports whether language is supported as a language.
// It panics if language is not valid.
func (fn *function) SupportLanguage(language state.Language) bool {
	switch language {
	case state.JavaScript:
		return fn.settings.Node.Runtime != ""
	case state.Python:
		return fn.settings.Python.Runtime != ""
	}
	panic("invalid language")
}

// Update updates the source of the function with the given identifier and
// returns a new version, which has a length in the range [1, 128].
// If the function does not exist, it returns the ErrFunctionNotExist error.
func (fn *function) Update(ctx context.Context, id, source string) (string, error) {
	arn, language, err := parseID(id)
	if err != nil {
		return "", err
	}
	code, err := fn.code(source, language)
	if err != nil {
		return "", err
	}
	client, err := fn.connect(ctx)
	if err != nil {
		return "", err
	}
	out, err := client.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
		FunctionName: &arn,
		Publish:      true,
		ZipFile:      code,
	})
	if err != nil {
		if status, ok := httpStatusCode(err); ok && status == 404 {
			return "", transformers.ErrFunctionNotExist
		}
		return "", err
	}
	if len(*out.Version) > 128 {
		return "", fmt.Errorf("transformers/lambda: version %q is too long", *out.Version)
	}
	return *out.Version, nil
}

// code returns the code of the function with the given source and language.
func (fn *function) code(source string, language state.Language) ([]byte, error) {
	var filename string
	var fullSource string
	switch language {
	case state.JavaScript:
		filename = "index.mjs"
		escapedSource := escapeJavaScriptSourceCode(source)
		fullSource = `
var transform;
export const _handler = async (event) => {
` + embed.JavaScriptNormalizeFunc + `
	if ( transform == null ) {
		try {
			Function(` + "`" + escapedSource + "`" + `);
		} catch (error) {
			return { error: error.toString() };
		}
		transform = Function('event', ` + "`" + escapedSource + "; return transform(event)`" + `);
	}
	event = Function("return " + event)();
	const records = [];
	for ( let i = 0; i < event.length; i++ ) {
		try {
			let value = transform(event[i]);
			normalize(value);
			records[i] = { value: value };
		} catch (error) {
			if (error instanceof Error) {
				error = error.toString();
			} else {
				error = "throw error of type " + (typeof error) + ": " + JSON.stringify(error);
			}
			records[i] = { error: error };
		}
	}
	return { records };
};
`
	case state.Python:
		filename = "index.py"
		fullSource = embed.PythonNormalizeFunc + "\n\n"
		fullSource += "_SOURCE = '''" + escapePythonSourceCode(source) + "'''\n\n"
		fullSource += `
def _handler(event, context):
	import json
	from uuid import UUID
	from decimal import Decimal
	from datetime import datetime, date, time

	try:
		exec(_SOURCE, globals())
	except SyntaxError as ex:
		error = f"SyntaxError: {ex.msg} (line {ex.lineno})"
		return json.dumps({"error": error}, separators=(",", ":"), default=str)
	except Exception as ex:
		name = type(ex).__name__
		error = f"{name}: {ex}"
		return json.dumps({"error": error}, separators=(",", ":"), default=str)
	records = []
	for e in eval(event):
		try:
			value = transform(e)
			_Norm.normalize(value)
		except Exception as ex:
			name = type(ex).__name__
			records.append({"error": f"{name}: {ex}"})
		else:
			records.append({"value": value})
	return json.dumps({"records": records}, separators=(",", ":"), default=str)
`
	}
	// Make a Zip file with the function code.
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	fi, err := zw.Create(filename)
	if err != nil {
		return nil, err
	}
	_, err = io.WriteString(fi, fullSource)
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// connect connects to Lambda and returns a client. If it is already connected,
// it returns the current client.
func (fn *function) connect(ctx context.Context) (*lambda.Client, error) {
	if fn.client != nil {
		return fn.client, nil
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(fn.settings.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(fn.settings.AccessKeyID, fn.settings.SecretAccessKey, "")))
	if err != nil {
		return nil, err
	}
	fn.client = lambda.NewFromConfig(cfg)
	return fn.client, nil
}

// pythonEscaper is used by escapePythonSourceCode.
//
// Keep this in sync with the code within the local transformer.
var pythonEscaper = strings.NewReplacer(`\`, `\\`, `'''`, `''\'`)

// escapePythonSourceCode escapes the given Python source code so it can be
// safely be put into a triple-quoted Python string literal (where the quote
// character is the single quote, not double) for later evaluation.
//
// Keep this in sync with the code within the local transformer.
func escapePythonSourceCode(src string) string {
	return pythonEscaper.Replace(src)
}

// javaScriptEscaper is used by escapeJavaScriptSourceCode.
//
// Keep this in sync with the code within the local transformer.
var javaScriptEscaper = strings.NewReplacer(`\`, `\\`, "`", "\\`", `$`, `\$`)

// escapeJavaScriptSourceCode escapes the given JavaScript source code so it can
// be safely be put into a single quoted JavaScript string literal for later
// evaluation.
//
// Keep this in sync with the code within the local transformer.
func escapeJavaScriptSourceCode(src string) string {
	return javaScriptEscaper.Replace(src)
}

// httpStatusCode returns the status code returned by a Lambda HTTP response.
// The boolean return value reports whether a status code exists.
func httpStatusCode(err error) (int, bool) {
	if err, ok := err.(*smithy.OperationError); ok {
		if err, ok := err.Err.(*http.ResponseError); ok {
			return err.Response.StatusCode, true
		}
	}
	return 0, false
}

// parseID parses a function identifier and returns its ARN and language.
func parseID(id string) (arn string, language state.Language, err error) {
	var ext string
	arn, ext, _ = strings.Cut(id, ".")
	switch ext {
	case "js":
		language = state.JavaScript
	case "py":
		language = state.Python
	default:
		return "", 0, fmt.Errorf("transformers/lambda: invalid function identifier %q", id)
	}
	return
}

// Metric error types.
const (
	errorTypeConnection = iota
	errorTypeNetwork
	errorTypeLambdaInternal
	errorTypeFunctionNotFound
	errorTypeSerialization
	errorTypeFunctionExec
)

var metricErrorLabels = [...]string{
	"connection",
	"network",
	"lambda_internal",
	"function_not_found",
	"serialization",
	"function_exec",
}

// Metrics.
var errorsMetric [len(metricErrorLabels)]*metrics.Counter
var durationMetric *metrics.Histogram
var recordsMetric *metrics.Counter

func init() {
	// Errors metric.
	vec := metrics.RegisterCounterVec(
		"meergo_lambda_errors_total",
		"Total number of Lambda errors, classified by type",
		[]string{"type"},
	)
	for i, status := range metricErrorLabels {
		errorsMetric[i] = vec.Register(status)
	}

	// Duration metric.
	durationMetric = metrics.RegisterHistogram(
		"meergo_lambda_duration_seconds",
		"Duration of successful Lambda executions in seconds",
		[]float64{0.1, 0.5, 1, 2.5, 5},
	)

	// Records metric.
	recordsMetric = metrics.RegisterCounter(
		"meergo_lambda_records_total",
		"Total number of input records processed by successful Lambda executions",
	)
}
