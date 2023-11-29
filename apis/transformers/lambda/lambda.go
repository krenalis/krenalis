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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"chichi/apis/state"
	"chichi/apis/transformers"
	"chichi/backoff"
	"chichi/connector/types"

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

// New returns a new function for Lambda with the given settings.
// Supports every Node and Python 3 runtime.
func New(settings Settings) transformers.Function {
	return &function{settings: settings}
}

// Call calls the function with the given name and version for each value and
// returns the result of each invocation. Each element of values is supposed to
// conform to inSchema. Each result conforms to outSchema unless a
// transformation error occurred, and in that case, the error is stored in the
// Err field of the result.
//
// It returns the ErrFunctionNotExist error if the function does not exist, and
// a FunctionExecutionError error if the function execution fails.
func (fn *function) Call(ctx context.Context, name, version string, inSchema, outSchema types.Type, values []map[string]any) ([]transformers.Result, error) {

	if !transformers.ValidFunctionName(name) {
		return nil, errors.New("function name is not valid")
	}
	ext := path.Ext(name)
	var language state.Language
	switch ext {
	case ".js":
		language = state.JavaScript
	case ".py":
		language = state.Python
	default:
		return nil, errors.New("language is not supported")
	}

	client, err := fn.connect(ctx)
	if err != nil {
		return nil, err
	}

	// Marshal the values.
	payload := make([]byte, 0, 1024)
	payload = append(payload, '"')
	payload, err = transformers.Marshal(payload, inSchema, values, language)
	if err != nil {
		return nil, err
	}
	payload = append(payload, '"')

	// Invoke the function.
	var out *lambda.InvokeOutput
	name = lambdaFunctionName(name)
	bo := backoff.New(100)
	bo.SetCap(3 * time.Second)
	for bo.Next(ctx) {
		out, err = client.Invoke(ctx, &lambda.InvokeInput{
			FunctionName: &name,
			Payload:      payload,
			Qualifier:    &version,
		})
		status, ok := httpStatusCode(err)
		if !ok {
			return nil, err
		}
		if status == 404 {
			return nil, transformers.ErrFunctionNotExist
		}
		if status == 409 {
			// The function is pending.
			// Set the base with a greater value and retry.
			bo.SetBase(300)
			continue
		}
		if 500 <= status && status <= 599 {
			// There was an internal error.
			// Set the base with the default value and retry.
			bo.SetBase(100)
			continue
		}
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}

	// Unmarshal the results.
	if out.FunctionError != nil {
		dec := json.NewDecoder(bytes.NewReader(out.Payload))
		payload := struct {
			ErrorMessage string
		}{}
		err = dec.Decode(&payload)
		if err != nil {
			return nil, fmt.Errorf("transformers/lambda: cannot decode response executing function %q: %s", name, err)
		}
		return nil, transformers.FunctionExecutionError(payload.ErrorMessage)
	}
	var r io.Reader
	switch ext {
	case ".js":
		r = bytes.NewReader(out.Payload)
	case ".py":
		var s string
		err = json.Unmarshal(out.Payload, &s)
		if err != nil {
			return nil, fmt.Errorf("transformers/lambda: cannot decode response executing function %q: %s", name, err)
		}
		r = strings.NewReader(s)
	}
	results, err := transformers.Unmarshal(r, outSchema, language)
	if err != nil {
		return nil, err
	}
	if len(results) != len(values) {
		return nil, fmt.Errorf("transformers/lambda: expected %d results from function %q, got %d", len(values), name, len(results))
	}
	return results, nil
}

// Close closes the function.
func (fn *function) Close(ctx context.Context) error {
	fn.client = nil
	return nil
}

// Create creates a new function with the given name and source, and returns its
// version, which has a length in the range [1, 128]. name should have an
// extension of either ".js" or ".py" depending on the source code's language.
// If a function with the same name already exists, it returns the
// ErrFunctionExist error.
func (fn *function) Create(ctx context.Context, name, source string) (string, error) {
	if !transformers.ValidFunctionName(name) {
		return "", errors.New("function name is not valid")
	}
	ext := path.Ext(name)
	if !fn.supportLanguage(ext) {
		return "", errors.New("language is not supported")
	}
	code, err := fn.code(source, ext)
	if err != nil {
		return "", err
	}
	client, err := fn.connect(ctx)
	if err != nil {
		return "", err
	}
	var runtime string
	var layers []string
	switch ext {
	case ".js":
		runtime = fn.settings.Node.Runtime
		if layer := fn.settings.Node.Layer; layer != "" {
			layers = []string{layer}
		}
	case ".py":
		runtime = fn.settings.Python.Runtime
		if layer := fn.settings.Python.Layer; layer != "" {
			layers = []string{layer}
		}
	}
	out, err := client.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(lambdaFunctionName(name)),
		Handler:      aws.String("index._handler"),
		Publish:      true,
		Role:         aws.String(fn.settings.Role),
		Runtime:      lambdatypes.Runtime(runtime),
		Code:         &lambdatypes.FunctionCode{ZipFile: code},
		Layers:       layers,
	})
	if err != nil {
		if status, ok := httpStatusCode(err); ok && status == 409 {
			return "", transformers.ErrFunctionExist
		}
		return "", err
	}
	if len(*out.Version) > 128 {
		return "", fmt.Errorf("transformers/lambda: version %q is too long", *out.Version)
	}
	return *out.Version, nil
}

// Delete deletes the function with the given name.
// If a function with the given name does not exist, it does nothing.
func (fn *function) Delete(ctx context.Context, name string) error {
	if !transformers.ValidFunctionName(name) {
		return errors.New("function name is not valid")
	}
	if !fn.supportLanguage(path.Ext(name)) {
		return errors.New("language is not supported")
	}
	client, err := fn.connect(ctx)
	if err != nil {
		return err
	}
	name = lambdaFunctionName(name)
	_, err = client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: &name,
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

// Update updates the source of the function with the given name, and returns a
// new version, which has a length in the range [1, 128]. If the function does
// not exist, it returns the ErrFunctionNotExist error.
func (fn *function) Update(ctx context.Context, name, source string) (string, error) {
	if !transformers.ValidFunctionName(name) {
		return "", errors.New("function name is not valid")
	}
	ext := path.Ext(name)
	if !fn.supportLanguage(ext) {
		return "", errors.New("language is not supported")
	}
	code, err := fn.code(source, ext)
	if err != nil {
		return "", err
	}
	client, err := fn.connect(ctx)
	if err != nil {
		return "", err
	}
	name = lambdaFunctionName(name)
	out, err := client.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
		FunctionName: &name,
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

// code returns the code of the function with the given source.
func (fn *function) code(source string, ext string) ([]byte, error) {
	var filename string
	switch ext {
	case ".js":
		filename = "index.mjs"
		source += `
BigInt.prototype.toJSON = function() { return this.toString(); }
export const _handler = async (event) => {
	event = Function("return " + event)();
	const results = [];
	for ( let i = 0; i < event.length; i++ ) {
		try {
			let value = transform(event[i]);
			results[i] = { value: value };
		} catch (error) {
			if (error instanceof Error) {
				error = error.toString();
			} else {
				error = "throw error of type " + (typeof error) + ": " + JSON.stringify(error);
			}
			results[i] = { error: error };
		}
	}
	return results;
};
`
	case ".py":
		filename = "index.py"
		source += `
def _handler(event, context):
	import json
	from uuid import UUID
	from decimal import Decimal
	from datetime import datetime, date, time
    
	results = []
	for e in eval(event):
		try:
			value = transform(e)
		except Exception as ex:
			name = type(ex).__name__
			results.append({"error": f"{name}: {ex}"})
		else:
			results.append({"value": value})
	return json.dumps(results, separators=(",", ":"), default=str)
`
	}
	// Make a Zip file with the function code.
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	fi, err := zw.Create(filename)
	if err != nil {
		return nil, err
	}
	_, err = io.WriteString(fi, source)
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

// supportLanguage is like SupportLanguage but gets an extension as argument.
func (fn *function) supportLanguage(ext string) bool {
	switch ext {
	case ".js":
		return fn.settings.Node.Runtime != ""
	case ".py":
		return fn.settings.Python.Runtime != ""
	}
	panic("invalid extension")
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

// lambdaFunctionName returns a function name in the format accepted by Lambda.
func lambdaFunctionName(name string) string {
	return name[:len(name)-3] + "_" + name[len(name)-2:]
}
