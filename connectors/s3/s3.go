//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

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
	"errors"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Connector icon.
var icon = "<svg></svg>"

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	meergo.RegisterFileStorage(meergo.FileStorageInfo{
		Name:       "S3",
		Categories: meergo.CategoryFileStorage,
		AsSource: &meergo.AsFileStorageSource{
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsFileStorageDestination{
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: destinationOverview,
			},
		},
		Icon: icon,
	}, New)
}

// New returns a new connector instance for S3.
func New(env *meergo.FileStorageEnv) (*S3, error) {
	c := S3{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for S3")
		}
	}
	return &c, nil
}

type S3 struct {
	env      *meergo.FileStorageEnv
	settings *innerSettings
}

type innerSettings struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
}

// AbsolutePath returns the absolute representation of the given path name.
func (ss3 *S3) AbsolutePath(ctx context.Context, name string) (string, error) {
	if len(name) > 1024 {
		return "", meergo.InvalidPathErrorf("path name cannot be longer than 1024 bytes")
	}
	if name[0] == '/' {
		name = name[1:]
	}
	return "s3://" + ss3.settings.Bucket + "/" + name, nil
}

// Reader opens a file and returns a ReadCloser from which to read its content.
func (ss3 *S3) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	if len(name) > 1024 {
		return nil, time.Time{}, meergo.NewInvalidSettingsError("object key cannot be longer than 1024 bytes")
	}
	client := ss3.client()
	res, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(ss3.settings.Bucket),
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
func (ss3 *S3) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if ss3.settings != nil {
			s = *ss3.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, ss3.saveSettings(ctx, settings)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "AccessKeyID", Label: "Access Key ID", Placeholder: "", Type: "text", MinLength: 20, MaxLength: 20},
			&meergo.Input{Name: "SecretAccessKey", Label: "Secret Access Key", Placeholder: "", Type: "password", MinLength: 40, MaxLength: 200},
			&meergo.Select{Name: "Region", Label: "Region", Placeholder: "", Options: []meergo.Option{
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
			&meergo.Input{Name: "Bucket", Label: "Bucket name", Placeholder: "mybucket", Type: "text", MinLength: 3, MaxLength: 63},
		},
		Settings: settings,
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (ss3 *S3) Write(ctx context.Context, p io.Reader, name, contentType string) error {
	if len(name) > 1024 {
		return meergo.NewInvalidSettingsError("object key cannot be longer than 1024 bytes")
	}
	if name[0] == '/' {
		name = name[1:]
	}
	client := ss3.client()
	u := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 8 * 1024 * 1024
		u.Concurrency = 2
	})
	_, err := u.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(ss3.settings.Bucket),
		Key:         aws.String(name),
		Body:        p,
		ContentType: aws.String(contentType),
	})
	return err
}

// client returns a S3 client.
func (ss3 *S3) client() *s3.Client {
	cfg := aws.Config{
		Region: ss3.settings.Region,
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(
				ss3.settings.AccessKeyID,
				ss3.settings.SecretAccessKey,
				"",
			),
		),
	}
	return s3.NewFromConfig(cfg)
}

// saveSettings validates and saves the settings.
func (ss3 *S3) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate AccessKeyID.
	if n := len(s.AccessKeyID); n != 20 {
		return meergo.NewInvalidSettingsError("access key id must be 20 bytes long")
	}
	// Validate SecretAccessKey.
	if n := len(s.SecretAccessKey); n < 40 || n > 200 {
		return meergo.NewInvalidSettingsError("secret access key length in bytes must be in range [40,200]")
	}
	// Validate Region.
	const regions = "us-east-1 us-east-2 us-west-1 us-west-2 af-south-1 ap-east-1 ap-south-2 ap-southeast-3 ap-southeast-5 " +
		"ap-southeast-4 ap-south-1 ap-northeast-3 ap-northeast-2 ap-southeast-1 ap-southeast-2 ap-east-2 ap-southeast-7 " +
		"ap-northeast-1 ca-central-1 ca-west-1 eu-central-1 eu-west-1 eu-west-2 eu-south-1 eu-west-3 eu-south-2 eu-north-1 " +
		"eu-central-2 il-central-1 mx-central-1 me-south-1 me-central-1 sa-east-1"
	if strings.Contains(s.Region, " ") || !strings.Contains(regions, s.Region+" ") && !strings.HasSuffix(regions, " "+s.Region) {
		return meergo.NewInvalidSettingsError("region is not valid")
	}
	// Validate Bucket.
	if n := len(s.Bucket); n < 3 || n > 63 {
		return meergo.NewInvalidSettingsError("bucket length must be in range [3,63]")
	}
	if !bucketReg.MatchString(s.Bucket) || strings.Contains(s.Bucket, "..") ||
		strings.HasPrefix(s.Bucket, "xn--") || strings.HasSuffix(s.Bucket, "-s3alias") {
		return meergo.NewInvalidSettingsError("bucket value is not allowed")
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ss3.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	ss3.settings = &s
	return nil
}
