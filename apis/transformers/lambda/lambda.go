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

type transformer struct {
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

// New returns a new Transformer for Lambda with the given settings.
// Supports every Node and Python 3 runtime.
func New(settings Settings) transformers.Transformer {
	return &transformer{settings: settings}
}

// CallFunction calls the function with the given name and version, with the
// given values to transform, and returns the results. inSchema and outSchema
// are the input and output schemas.
//
// If an error occurs during execution, it returns an *ExecutionError error. If
// the function does not exist, it returns the ErrNotExist error. If the
// function is in a pending state, it returns the ErrPendingState error.
func (tr *transformer) CallFunction(ctx context.Context, name, version string, inSchema, outSchema types.Type, values []map[string]any) ([]transformers.Result, error) {

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

	client, err := tr.connect(ctx)
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
	bo := backoff.New(10, 10, 1*time.Second)
	for bo.Next(ctx) {
		out, err = client.Invoke(ctx, &lambda.InvokeInput{
			FunctionName: &name,
			Payload:      payload,
			Qualifier:    &version,
		})
		if isHTTPErrorCode(err, 409) {
			if bo.Attempt() == 1 {
				bo.SetNextWaitTime(3 * time.Second)
			}
			continue
		}
		break
	}
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			return nil, transformers.ErrNotExist
		}
		if isHTTPErrorCode(err, 409) {
			return nil, transformers.ErrPendingState
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
		return nil, transformers.NewExecutionError(payload.ErrorMessage)
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

// Close closes the transformer.
func (tr *transformer) Close(ctx context.Context) error {
	tr.client = nil
	return nil
}

// CreateFunction creates a new function with the given name and source, and
// returns its version, which has a length in the range [1, 128]. name should
// have an extension of either ".js" or ".py" depending on the source code's
// language. If a function with the same name already exists, it returns the
// ErrExist error.
func (tr *transformer) CreateFunction(ctx context.Context, name, source string) (string, error) {
	if !transformers.ValidFunctionName(name) {
		return "", errors.New("function name is not valid")
	}
	ext := path.Ext(name)
	if !tr.supportLanguage(ext) {
		return "", errors.New("language is not supported")
	}
	code, err := tr.code(source, ext)
	if err != nil {
		return "", err
	}
	client, err := tr.connect(ctx)
	if err != nil {
		return "", err
	}
	var runtime string
	var layers []string
	switch ext {
	case ".js":
		runtime = tr.settings.Node.Runtime
		if layer := tr.settings.Node.Layer; layer != "" {
			layers = []string{layer}
		}
	case ".py":
		runtime = tr.settings.Python.Runtime
		if layer := tr.settings.Python.Layer; layer != "" {
			layers = []string{layer}
		}
	}
	out, err := client.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(lambdaFunctionName(name)),
		Handler:      aws.String("index._handler"),
		Publish:      true,
		Role:         aws.String(tr.settings.Role),
		Runtime:      lambdatypes.Runtime(runtime),
		Code:         &lambdatypes.FunctionCode{ZipFile: code},
		Layers:       layers,
	})
	if err != nil {
		if isHTTPErrorCode(err, 409) {
			return "", transformers.ErrExist
		}
		return "", err
	}
	if len(*out.Version) > 128 {
		return "", fmt.Errorf("transformers/lambda: version %q is too long", *out.Version)
	}
	return *out.Version, nil
}

// DeleteFunction deletes the function with the given name.
// If a function with the given name does not exist, it does nothing.
func (tr *transformer) DeleteFunction(ctx context.Context, name string) error {
	if !transformers.ValidFunctionName(name) {
		return errors.New("function name is not valid")
	}
	if !tr.supportLanguage(path.Ext(name)) {
		return errors.New("language is not supported")
	}
	client, err := tr.connect(ctx)
	if err != nil {
		return err
	}
	name = lambdaFunctionName(name)
	_, err = client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: &name,
	})
	if err != nil && isHTTPErrorCode(err, 404) {
		err = nil
	}
	return err
}

// SupportLanguage reports whether language is supported as a language.
// It panics if language is not valid.
func (tr *transformer) SupportLanguage(language state.Language) bool {
	switch language {
	case state.JavaScript:
		return tr.settings.Node.Runtime != ""
	case state.Python:
		return tr.settings.Python.Runtime != ""
	}
	panic("invalid language")
}

// UpdateFunction updates the source of the function with the given name, and
// returns a new version, which has a length in the range [1, 128]. If the
// function does not exist, it returns the ErrNotExist error.
func (tr *transformer) UpdateFunction(ctx context.Context, name, source string) (string, error) {
	if !transformers.ValidFunctionName(name) {
		return "", errors.New("function name is not valid")
	}
	ext := path.Ext(name)
	if !tr.supportLanguage(ext) {
		return "", errors.New("language is not supported")
	}
	code, err := tr.code(source, ext)
	if err != nil {
		return "", err
	}
	client, err := tr.connect(ctx)
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
		if isHTTPErrorCode(err, 404) {
			return "", transformers.ErrNotExist
		}
		return "", err
	}
	if len(*out.Version) > 128 {
		return "", fmt.Errorf("transformers/lambda: version %q is too long", *out.Version)
	}
	return *out.Version, nil
}

// code returns the code of the function with the given source.
func (tr *transformer) code(source string, ext string) ([]byte, error) {
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
			results.append({"error": str(ex)})
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
func (tr *transformer) connect(ctx context.Context) (*lambda.Client, error) {
	if tr.client != nil {
		return tr.client, nil
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(tr.settings.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(tr.settings.AccessKeyID, tr.settings.SecretAccessKey, "")))
	if err != nil {
		return nil, err
	}
	tr.client = lambda.NewFromConfig(cfg)
	return tr.client, nil
}

// supportLanguage is like SupportLanguage but gets an extension as argument.
func (tr *transformer) supportLanguage(ext string) bool {
	switch ext {
	case ".js":
		return tr.settings.Node.Runtime != ""
	case ".py":
		return tr.settings.Python.Runtime != ""
	}
	panic("invalid extension")
}

// isHTTPErrorCode checks whether an error relates to an HTTP response error and
// if the HTTP status code matches the specified code.
func isHTTPErrorCode(err error, code int) bool {
	if err, ok := err.(*smithy.OperationError); ok {
		if err, ok := err.Err.(*http.ResponseError); ok {
			return err.Response.StatusCode == code
		}
	}
	return false
}

// lambdaFunctionName returns a function name in the format accepted by Lambda.
func lambdaFunctionName(name string) string {
	return name[:len(name)-3] + "_" + name[len(name)-2:]
}
