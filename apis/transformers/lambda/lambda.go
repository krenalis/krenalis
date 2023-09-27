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
	"fmt"
	"io"
	"strings"

	"chichi/apis/transformers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
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
	Runtime         string
	Layer           string
}

// New returns a new Transformer for Lambda with the given settings.
// Supports every Node and Python 3 runtime.
func New(settings Settings) transformers.Transformer {
	return &transformer{settings: settings}
}

// CallFunction calls the function with the given name and version, with the
// given values to transform, and returns the results. If an error occurs during
// execution, it returns an ExecutionError. If the function does not exist, it
// returns the ErrNotExist error. If the function is in a pending state, it
// returns the ErrPendingState error.
func (tr *transformer) CallFunction(ctx context.Context, name, version string, values []map[string]any) ([]transformers.Result, error) {

	// Create a client.
	client, err := tr.connect(ctx)
	if err != nil {
		return nil, err
	}

	// Marshal the values.
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err = enc.Encode(values)
	if err != nil {
		return nil, err
	}

	// Invoke the function.
	out, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: &name,
		Payload:      b.Bytes(),
		Qualifier:    &version,
	})
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			return nil, transformers.ErrNotExist
		}
		if isHTTPErrorCode(err, 409) {
			return nil, transformers.ErrPendingState
		}
		return nil, err
	}

	// Unmarshal the results.
	dec := json.NewDecoder(bytes.NewReader(out.Payload))
	dec.UseNumber()
	if out.FunctionError != nil {
		payload := struct {
			ErrorMessage string
		}{}
		err = dec.Decode(&payload)
		if err != nil {
			return nil, fmt.Errorf("transformers/lambda: cannot decode response executing function %q: %s", name, err)
		}
		return nil, transformers.NewExecutionError(payload.ErrorMessage)
	}
	results := make([]transformers.Result, 0, len(values))
	err = dec.Decode(&results)
	if err != nil {
		return nil, fmt.Errorf("transformers/lambda: cannot decode response executing function %q: %s", name, err)
	}
	if len(results) != len(values) {
		return nil, fmt.Errorf("transformers/lambda: expected %d results from function %q, got %d", len(values), name, len(results))
	}
	for _, r := range results {
		if (r.Value == nil) == (r.Error == "") {
			return nil, fmt.Errorf("transformers/lambda: invalid results from function %q", name)
		}
	}

	return results, nil
}

// Close closes the transformer.
func (tr *transformer) Close(ctx context.Context) error {
	tr.client = nil
	return nil
}

// CreateFunction creates a new function with the given name and source, and
// returns its version, which has a length in the range [1, 128]. If a function
// with the same name already exists, it returns the ErrExist error.
func (tr *transformer) CreateFunction(ctx context.Context, name, source string) (string, error) {
	code, err := tr.code(source)
	if err != nil {
		return "", err
	}
	client, err := tr.connect(ctx)
	if err != nil {
		return "", err
	}
	var layers []string
	if tr.settings.Layer != "" {
		layers = []string{tr.settings.Layer}
	}
	out, err := client.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(name),
		Handler:      aws.String("index._handler"),
		Publish:      true,
		Role:         aws.String(tr.settings.Role),
		Runtime:      types.Runtime(tr.settings.Runtime),
		Code:         &types.FunctionCode{ZipFile: code},
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
	client, err := tr.connect(ctx)
	if err != nil {
		return err
	}
	_, err = client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: &name,
	})
	if err != nil && isHTTPErrorCode(err, 404) {
		err = nil
	}
	return err
}

// UpdateFunction updates the source of the function with the given name, and
// returns a new version, which has a length in the range [1, 128]. If the
// function does not exist, it returns the ErrNotExist error.
func (tr *transformer) UpdateFunction(ctx context.Context, name, source string) (string, error) {
	code, err := tr.code(source)
	if err != nil {
		return "", err
	}
	client, err := tr.connect(ctx)
	if err != nil {
		return "", err
	}
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
func (tr *transformer) code(source string) ([]byte, error) {
	var filename string
	switch {
	case strings.HasPrefix(tr.settings.Runtime, "nodejs"):
		filename = "index.mjs"
		source += `
export const _handler = async (event) => {
	const results = [];
	for ( let i = 0; i < event.length; i++ ) {
		try {
			let value = transform(event[i]);
			results[i] = { "value": value };
		} catch (error) {
			results[i] = { "error": error };
		}
	}
	return results;
};
`
	case strings.HasPrefix(tr.settings.Runtime, "python3."):
		filename = "index.py"
		source += `
def _handler(event, context):
	results = []
	for e in event:
		try:
			value = transform(e)
		except Exception as ex:
			results.append({"error": str(ex)})
		else:
			results.append({"value": value})
	return results
`
	default:
		return nil, fmt.Errorf("invalid runtime %q", tr.settings.Runtime)
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
