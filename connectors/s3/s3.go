//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package s3

// This package is the S3 connector.
// (https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html)

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis"
	"chichi/connector"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Make sure it implements the StreamConnection interface.
var _ connector.StreamConnection = &connection{}

func init() {
	apis.RegisterStreamConnector("S3", New)
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
	ObjectKey       string
	ContentType     string
}

// New returns a new S3 connection.
func New(ctx context.Context, settings []byte, fh connector.Firehose) (connector.StreamConnection, error) {
	c := connection{ctx: ctx}
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of S3 connection")
		}
	}
	c.firehose = fh
	return &c, nil
}

// Reader returns a ReadCloser from which to read the data and its last update
// time.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Reader() (io.ReadCloser, time.Time, error) {
	client := c.client()
	res, err := client.GetObject(c.ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.settings.Bucket),
		Key:    aws.String(c.settings.ObjectKey),
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
func (c *connection) ServeUI(event string, form []byte) (*connector.SettingsUI, error) {

	var s settings

	switch event {
	case "load":
		// Load the UI.
		if c.settings != nil {
			s = *c.settings
		}
	case "save":
		// Save the settings.
		err := json.Unmarshal(form, &s)
		if err != nil {
			return nil, err
		}
		// Validate SecretAccessKey.
		if n := len(s.SecretAccessKey); n != 20 {
			return nil, connector.UIErrorf("access key id must be 20 characters long")
		}
		// Validate AccessKeyID.
		if n := len(s.AccessKeyID); n < 50 || n > 200 {
			return nil, connector.UIErrorf("secret access key length must be in range [50,200]")
		}
		// Validate Region.
		const regions = "us-east-1 us-east-2 us-west-1 us-west-2 af-south-1 ap-east-1 ap-southeast-3 ap-south-1 " +
			"ap-northeast-1 ap-northeast-2 ap-northeast-3 ap-southeast-1 ap-southeast-2 ca-central-1 eu-central-1 " +
			"eu-west-1 eu-west-2 eu-west-3 eu-south-1 eu-north-1 me-south-1 me-central-1 sa-east-1"
		if !strings.Contains(regions, s.Region+" ") || !strings.HasSuffix(regions, " "+s.Region) {
			return nil, connector.UIErrorf("region is not valid")
		}
		// Validate Bucket.
		if n := len(s.Bucket); n < 3 || n > 63 {
			return nil, connector.UIErrorf("bucket length must be in range [3,63]")
		}
		if !bucketReg.MatchString(s.Bucket) || strings.Contains(s.Bucket, "..") ||
			strings.HasPrefix(s.Bucket, "xn--") || strings.HasSuffix(s.Bucket, "-s3alias") {
			return nil, connector.UIErrorf("bucket value is not allowed")
		}
		// Validate ObjectKey.
		if n := len(s.ObjectKey); n == 0 || n > 1024 {
			return nil, connector.UIErrorf("object key length in bytes must be in range [1,1024]")
		}
		// Validate ContentType.
		if n := utf8.RuneCountInString(s.ContentType); n < 3 || n > 100 {
			return nil, connector.UIErrorf("content type length must be in range [3,100]")
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	default:
		return nil, connector.ErrEventNotExist
	}

	ui := &connector.SettingsUI{
		Components: []connector.Component{
			&connector.Input{Name: "accessKeyID", Value: s.AccessKeyID, Label: "Access Key ID", Placeholder: "Access Key ID", Type: "text", MinLength: 20, MaxLength: 20},
			&connector.Input{Name: "secretAccessKey", Value: s.SecretAccessKey, Label: "Secret Access Key", Placeholder: "Secret Access Key", Type: "password", MinLength: 50, MaxLength: 200},
			&connector.Select{Name: "region", Value: s.Region, Label: "Region", Placeholder: "Region", Options: []connector.Option{
				{Text: "US East (N. Virginia) us-east-1", Value: "us-east-1"},
				{Text: "US East (Ohio) us-east-2", Value: "us-east-2"},
				{Text: "US West (N. California) us-west-1", Value: "us-west-1"},
				{Text: "US West (Oregon) us-west-2", Value: "us-west-2"},
				{Text: "Africa (Cape Town) af-south-1", Value: "af-south-1"},
				{Text: "Asia Pacific (Hong Kong) ap-east-1", Value: "ap-east-1"},
				{Text: "Asia Pacific (Jakarta) ap-southeast-3", Value: "ap-southeast-3"},
				{Text: "Asia Pacific (Mumbai) ap-south-1", Value: "ap-south-1"},
				{Text: "Asia Pacific (Osaka) ap-northeast-3", Value: "ap-northeast-3"},
				{Text: "Asia Pacific (Seoul) ap-northeast-2", Value: "ap-northeast-2"},
				{Text: "Asia Pacific (Singapore) ap-southeast-1", Value: "ap-southeast-1"},
				{Text: "Asia Pacific (Sydney) ap-southeast-2", Value: "ap-southeast-2"},
				{Text: "Asia Pacific (Tokyo) ap-northeast-1", Value: "ap-northeast-1"},
				{Text: "Canada (Central) ap-northeast-1", Value: "ca-central-1"},
				{Text: "Europe (Frankfurt) eu-central-1", Value: "eu-central-1"},
				{Text: "Europe (Ireland) eu-west-1", Value: "eu-west-1"},
				{Text: "Europe (London) eu-west-2", Value: "eu-west-2"},
				{Text: "Europe (Milan) eu-south-1", Value: "eu-south-1"},
				{Text: "Europe (Paris) eu-west-3", Value: "eu-west-3"},
				{Text: "Europe (Stockholm) eu-north-1", Value: "eu-north-1"},
				{Text: "Middle East (Bahrain) me-south-1", Value: "me-south-1"},
				{Text: "Middle East (UAE) me-central-1", Value: "me-central-1"},
				{Text: "South America (São Paulo) me-central-1", Value: "sa-east-1"},
			}},
			&connector.Input{Name: "bucket", Value: s.Bucket, Label: "Bucket Name", Placeholder: "bucket", Type: "text", MinLength: 3, MaxLength: 63},
			&connector.Input{Name: "objectKey", Value: s.ObjectKey, Label: "Object Key", Placeholder: "users.csv", Type: "text", MinLength: 1, MaxLength: 1024},
			&connector.Input{Name: "contentType", Value: s.ContentType, Label: "Content Type", Placeholder: "text/csv", Type: "text", MinLength: 3, MaxLength: 100},
		},
		Actions: []connector.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return ui, nil
}

// Write writes the data read from p.
func (c *connection) Write(p io.Reader) error {
	client := c.client()
	_, err := client.PutObject(c.ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.settings.Bucket),
		Key:         aws.String(c.settings.ObjectKey),
		Body:        p,
		ContentType: &c.settings.ContentType,
	})
	return err
}

// client returns a S3 client.
// (https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/).
func (c *connection) client() *s3.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(c.settings.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(c.settings.AccessKeyID, c.settings.SecretAccessKey, "")))
	if err != nil {
		return nil
	}
	return s3.NewFromConfig(cfg)
}
