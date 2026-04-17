// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package s3 provides a connector for S3.
// (https://docs.aws.amazon.com/AmazonS3/latest/API/)
//
// S3 is a trademark of Amazon Technologies, Inc.
// This connector is not affiliated with or endorsed by Amazon Technologies,
// Inc.
package s3

import (
	"context"
	_ "embed"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	s3pkg "github.com/aws/aws-sdk-go-v2/service/s3"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterFileStorage(connectors.FileStorageSpec{
		Code:       "s3",
		Label:      "S3",
		Categories: connectors.CategoryFileStorage,
		AsSource: &connectors.AsFileStorageSource{
			Documentation: connectors.RoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsFileStorageDestination{
			Documentation: connectors.RoleDocumentation{
				Overview: destinationOverview,
			},
		},
	}, New)
}

// New returns a new connector instance for S3.
func New(env *connectors.FileStorageEnv) (*S3, error) {
	return &S3{env: env}, nil
}

type S3 struct {
	env *connectors.FileStorageEnv
}

type innerSettings struct {
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
}

// AbsolutePath returns the absolute representation of the given path name.
func (s3 *S3) AbsolutePath(ctx context.Context, name string) (string, error) {
	if len(name) > 1024 {
		return "", connectors.InvalidPathErrorf("path name cannot be longer than 1024 bytes")
	}
	if name[0] == '/' {
		name = name[1:]
	}
	var s innerSettings
	err := s3.env.Settings.Load(ctx, &s)
	if err != nil {
		return "", err
	}
	return "s3://" + s.Bucket + "/" + name, nil
}

// Reader opens a file and returns a ReadCloser from which to read its content.
func (s3 *S3) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	if len(name) > 1024 {
		return nil, time.Time{}, connectors.NewInvalidSettingsError("object key cannot be longer than 1024 bytes")
	}
	var s innerSettings
	err := s3.env.Settings.Load(ctx, &s)
	if err != nil {
		return nil, time.Time{}, err
	}
	client := s3.client(&s)
	res, err := client.GetObject(ctx, &s3pkg.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(name),
	})
	if err != nil {
		return nil, time.Time{}, err
	}
	var ts time.Time
	if res.LastModified == nil {
		ts = time.Now()
	} else {
		ts = *res.LastModified
	}
	return res.Body, ts.UTC(), nil
}

var bucketReg = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]+$`)

// ServeUI serves the connector's user interface.
func (s3 *S3) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		err := s3.env.Settings.Load(ctx, &s)
		if err != nil {
			return nil, err
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, s3.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "accessKeyID", Label: "Access Key ID", Placeholder: "", Type: "text", MinLength: 20, MaxLength: 20},
			&connectors.Input{Name: "secretAccessKey", Label: "Secret Access Key", Placeholder: "", Type: "password", MinLength: 40, MaxLength: 200},
			&connectors.Select{Name: "region", Label: "Region", Placeholder: "", Options: []connectors.Option{
				{Text: "US East (N. Virginia) us-east-1", Value: "us-east-1"},
				{Text: "US East (Ohio) us-east-2", Value: "us-east-2"},
				{Text: "US West (N. California) us-west-1", Value: "us-west-1"},
				{Text: "US West (Oregon) us-west-2", Value: "us-west-2"},
				{Text: "Africa (Cape Town) af-south-1", Value: "af-south-1"},
				{Text: "Asia Pacific (Hong Kong) ap-east-1", Value: "ap-east-1"},
				{Text: "Asia Pacific (Hyderabad) ap-south-2", Value: "ap-south-2"},
				{Text: "Asia Pacific (Jakarta) ap-southeast-3", Value: "ap-southeast-3"},
				{Text: "Asia Pacific (Malaysia) ap-southeast-5", Value: "ap-southeast-5"},
				{Text: "Asia Pacific (Melbourne) ap-southeast-4", Value: "ap-southeast-4"},
				{Text: "Asia Pacific (Mumbai) ap-south-1", Value: "ap-south-1"},
				{Text: "Asia Pacific (Osaka) ap-northeast-3", Value: "ap-northeast-3"},
				{Text: "Asia Pacific (Seoul) ap-northeast-2", Value: "ap-northeast-2"},
				{Text: "Asia Pacific (Singapore) ap-southeast-1", Value: "ap-southeast-1"},
				{Text: "Asia Pacific (Sydney) ap-southeast-2", Value: "ap-southeast-2"},
				{Text: "Asia Pacific (Taipei) ap-east-2", Value: "ap-east-2"},
				{Text: "Asia Pacific (Thailand) ap-southeast-7", Value: "ap-southeast-7"},
				{Text: "Asia Pacific (Tokyo) ap-northeast-1", Value: "ap-northeast-1"},
				{Text: "Canada (Central) ca-central-1", Value: "ca-central-1"},
				{Text: "Canada West (Calgary) ca-west-1", Value: "ca-west-1"},
				{Text: "Europe (Frankfurt) eu-central-1", Value: "eu-central-1"},
				{Text: "Europe (Ireland) eu-west-1", Value: "eu-west-1"},
				{Text: "Europe (London) eu-west-2", Value: "eu-west-2"},
				{Text: "Europe (Milan) eu-south-1", Value: "eu-south-1"},
				{Text: "Europe (Paris) eu-west-3", Value: "eu-west-3"},
				{Text: "Europe (Spain) eu-south-2", Value: "eu-south-2"},
				{Text: "Europe (Stockholm) eu-north-1", Value: "eu-north-1"},
				{Text: "Europe (Zurich) eu-central-2", Value: "eu-central-2"},
				{Text: "Israel (Tel Aviv) il-central-1", Value: "il-central-1"},
				{Text: "Mexico (Central) mx-central-1", Value: "mx-central-1"},
				{Text: "Middle East (Bahrain) me-south-1", Value: "me-south-1"},
				{Text: "Middle East (UAE) me-central-1", Value: "me-central-1"},
				{Text: "South America (São Paulo) sa-east-1", Value: "sa-east-1"},
			}},
			&connectors.Input{Name: "bucket", Label: "Bucket name", Placeholder: "mybucket", Type: "text", MinLength: 3, MaxLength: 63},
		},
		Settings: settings,
		Buttons:  []connectors.Button{connectors.SaveButton},
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (s3 *S3) Write(ctx context.Context, p io.Reader, name, contentType string) error {
	if len(name) > 1024 {
		return connectors.NewInvalidSettingsError("object key cannot be longer than 1024 bytes")
	}
	var s innerSettings
	err := s3.env.Settings.Load(ctx, &s)
	if err != nil {
		return err
	}
	if name[0] == '/' {
		name = name[1:]
	}
	client := s3.client(&s)
	tm := transfermanager.New(client, func(opts *transfermanager.Options) {
		opts.PartSizeBytes = 8 * 1024 * 1024
		opts.Concurrency = 2
	})
	_, err = tm.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(name),
		Body:        p,
		ContentType: aws.String(contentType),
	})
	return err
}

// client returns a S3 client.
func (s3 *S3) client(s *innerSettings) *s3pkg.Client {
	cfg := aws.Config{
		Region: s.Region,
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(
				s.AccessKeyID,
				s.SecretAccessKey,
				"",
			),
		),
	}
	return s3pkg.NewFromConfig(cfg)
}

// saveSettings validates and saves the settings.
func (s3 *S3) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate AccessKeyID.
	if n := len(s.AccessKeyID); n != 20 {
		return connectors.NewInvalidSettingsError("access key id must be 20 bytes long")
	}
	// Validate SecretAccessKey.
	if n := len(s.SecretAccessKey); n < 40 || n > 200 {
		return connectors.NewInvalidSettingsError("secret access key length in bytes must be in range [40,200]")
	}
	// Validate Region.
	const regions = "us-east-1 us-east-2 us-west-1 us-west-2 af-south-1 ap-east-1 ap-south-2 ap-southeast-3 ap-southeast-5 " +
		"ap-southeast-4 ap-south-1 ap-northeast-3 ap-northeast-2 ap-southeast-1 ap-southeast-2 ap-east-2 ap-southeast-7 " +
		"ap-northeast-1 ca-central-1 ca-west-1 eu-central-1 eu-west-1 eu-west-2 eu-south-1 eu-west-3 eu-south-2 eu-north-1 " +
		"eu-central-2 il-central-1 mx-central-1 me-south-1 me-central-1 sa-east-1"
	if strings.Contains(s.Region, " ") || !strings.Contains(regions, s.Region+" ") && !strings.HasSuffix(regions, " "+s.Region) {
		return connectors.NewInvalidSettingsError("region is not valid")
	}
	// Validate Bucket.
	if n := len(s.Bucket); n < 3 || n > 63 {
		return connectors.NewInvalidSettingsError("bucket length must be in range [3,63]")
	}
	if !bucketReg.MatchString(s.Bucket) || strings.Contains(s.Bucket, "..") ||
		strings.HasPrefix(s.Bucket, "xn--") || strings.HasSuffix(s.Bucket, "-s3alias") {
		return connectors.NewInvalidSettingsError("bucket value is not allowed")
	}
	return s3.env.Settings.Store(ctx, s)
}
