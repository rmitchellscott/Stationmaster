package services

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/logging"
)

const (
	S3BucketURL = "https://trmnl-fw.s3.us-east-2.amazonaws.com"
)

type S3ListBucketResult struct {
	XMLName  xml.Name       `xml:"ListBucketResult"`
	Contents []S3ObjectInfo `xml:"Contents"`
}

type S3ObjectInfo struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
	Size         int64     `xml:"Size"`
	ETag         string    `xml:"ETag"`
}

type FirmwareInfo struct {
	Version      string
	DownloadURL  string
	ReleasedAt   time.Time
	FileSize     int64
	ETag         string
}

var firmwareFilenameRegex = regexp.MustCompile(`^FW([0-9.]+)\.bin$`)

func FetchS3FirmwareList(ctx context.Context) ([]FirmwareInfo, error) {
	logging.Info("[S3 FIRMWARE] Fetching firmware list from S3 bucket")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	url := fmt.Sprintf("%s/?list-type=2", S3BucketURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch S3 listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("S3 returned status %d", resp.StatusCode)
	}

	var listResult S3ListBucketResult
	if err := xml.NewDecoder(resp.Body).Decode(&listResult); err != nil {
		return nil, fmt.Errorf("failed to parse S3 XML response: %w", err)
	}

	var firmwareList []FirmwareInfo
	for _, obj := range listResult.Contents {
		matches := firmwareFilenameRegex.FindStringSubmatch(obj.Key)
		if len(matches) != 2 {
			continue
		}

		version := matches[1]
		firmware := FirmwareInfo{
			Version:     version,
			DownloadURL: fmt.Sprintf("%s/%s", S3BucketURL, obj.Key),
			ReleasedAt:  obj.LastModified,
			FileSize:    obj.Size,
			ETag:        obj.ETag,
		}
		firmwareList = append(firmwareList, firmware)
	}

	logging.Info("[S3 FIRMWARE] Found firmware versions in S3", "count", len(firmwareList))
	return firmwareList, nil
}
