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
	"net/http"
	"time"

	"chichi/connectors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Make sure it implements the StreamConnection interface.
var _ connectors.StreamConnection = &connection{}

func init() {
	connectors.RegisterStreamConnector("S3", New)
}

type connection struct {
	ctx      context.Context
	settings *settings
}

type settings struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
	Key             string
	ContentType     string
}

// New returns a new S3 connection.
func New(ctx context.Context, settings []byte, fh connectors.Firehose) (connectors.StreamConnection, error) {
	c := connection{ctx: ctx}
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of S3 connection")
		}
	}
	return &c, nil
}

// Reader returns a ReadCloser from which to read the data and its last update
// time.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Reader() (io.ReadCloser, time.Time, error) {
	client := c.client()
	res, err := client.GetObject(c.ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.settings.Bucket),
		Key:    aws.String(c.settings.Key),
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

// ServeUserInterface serves the connector's user interface.
func (c *connection) ServeUserInterface(w http.ResponseWriter, r *http.Request) {}

// Write writes the data read from p.
func (c *connection) Write(p io.Reader) error {
	client := c.client()
	_, err := client.PutObject(c.ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.settings.Bucket),
		Key:         aws.String(c.settings.Key),
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
