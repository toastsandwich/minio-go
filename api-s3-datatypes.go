/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2020 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package minio

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"reflect"
	"time"
)

// listAllMyBucketsResult container for listBuckets response.
type listAllMyBucketsResult struct {
	// Container for one or more buckets.
	Buckets struct {
		Bucket []BucketInfo
	}
	Owner owner
}

// listAllMyDirectoryBucketsResult container for listDirectoryBuckets response.
type listAllMyDirectoryBucketsResult struct {
	Buckets struct {
		Bucket []BucketInfo
	}
	ContinuationToken string
}

// owner container for bucket owner information.
type owner struct {
	DisplayName string
	ID          string
}

// CommonPrefix container for prefix response.
type CommonPrefix struct {
	Prefix string
}

// ListBucketV2Result container for listObjects response version 2.
type ListBucketV2Result struct {
	// A response can contain CommonPrefixes only if you have
	// specified a delimiter.
	CommonPrefixes []CommonPrefix
	// Metadata about each object returned.
	Contents  []ObjectInfo
	Delimiter string

	// Encoding type used to encode object keys in the response.
	EncodingType string

	// A flag that indicates whether or not ListObjects returned all of the results
	// that satisfied the search criteria.
	IsTruncated bool
	MaxKeys     int64
	Name        string

	// Hold the token that will be sent in the next request to fetch the next group of keys
	NextContinuationToken string

	ContinuationToken string
	Prefix            string

	// FetchOwner and StartAfter are currently not used
	FetchOwner string
	StartAfter string
}

// Version is an element in the list object versions response
type Version struct {
	ETag         string
	IsLatest     bool
	Key          string
	LastModified time.Time
	Owner        Owner
	Size         int64
	StorageClass string
	VersionID    string `xml:"VersionId"`

	// x-amz-meta-* headers stripped "x-amz-meta-" prefix containing the first value.
	// Only returned by MinIO servers.
	UserMetadata StringMap `json:"userMetadata,omitempty"`

	// x-amz-tagging values in their k/v values.
	// Only returned by MinIO servers.
	UserTags URLMap `json:"userTags,omitempty" xml:"UserTags"`

	Internal *struct {
		K int // Data blocks
		M int // Parity blocks
	} `xml:"Internal"`

	// Checksum values. Only returned by AiStor servers.
	ChecksumCRC32     string `xml:",omitempty"`
	ChecksumCRC32C    string `xml:",omitempty"`
	ChecksumSHA1      string `xml:",omitempty"`
	ChecksumSHA256    string `xml:",omitempty"`
	ChecksumCRC64NVME string `xml:",omitempty"`
	ChecksumType      string `xml:",omitempty"`

	isDeleteMarker bool
}

// ListVersionsResult is an element in the list object versions response
// and has a special Unmarshaler because we need to preserver the order
// of <Version>  and <DeleteMarker> in ListVersionsResult.Versions slice
type ListVersionsResult struct {
	Versions []Version

	CommonPrefixes      []CommonPrefix
	Name                string
	Prefix              string
	Delimiter           string
	MaxKeys             int64
	EncodingType        string
	IsTruncated         bool
	KeyMarker           string
	VersionIDMarker     string
	NextKeyMarker       string
	NextVersionIDMarker string
}

// UnmarshalXML is a custom unmarshal code for the response of ListObjectVersions, the custom
// code will unmarshal <Version> and <DeleteMarker> tags and save them in Versions field to
// preserve the lexical order of the listing.
func (l *ListVersionsResult) UnmarshalXML(d *xml.Decoder, _ xml.StartElement) (err error) {
	for {
		// Read tokens from the XML document in a stream.
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		se, ok := t.(xml.StartElement)
		if ok {
			tagName := se.Name.Local
			switch tagName {
			case "Name", "Prefix",
				"Delimiter", "EncodingType",
				"KeyMarker", "NextKeyMarker":
				var s string
				if err = d.DecodeElement(&s, &se); err != nil {
					return err
				}
				v := reflect.ValueOf(l).Elem().FieldByName(tagName)
				if v.IsValid() {
					v.SetString(s)
				}
			case "VersionIdMarker":
				// VersionIdMarker is a special case because of 'Id' instead of 'ID' in field name
				var s string
				if err = d.DecodeElement(&s, &se); err != nil {
					return err
				}
				l.VersionIDMarker = s
			case "NextVersionIdMarker":
				// NextVersionIdMarker is a special case because of 'Id' instead of 'ID' in field name
				var s string
				if err = d.DecodeElement(&s, &se); err != nil {
					return err
				}
				l.NextVersionIDMarker = s
			case "IsTruncated": //        bool
				var b bool
				if err = d.DecodeElement(&b, &se); err != nil {
					return err
				}
				l.IsTruncated = b
			case "MaxKeys": //       int64
				var i int64
				if err = d.DecodeElement(&i, &se); err != nil {
					return err
				}
				l.MaxKeys = i
			case "CommonPrefixes":
				var cp CommonPrefix
				if err = d.DecodeElement(&cp, &se); err != nil {
					return err
				}
				l.CommonPrefixes = append(l.CommonPrefixes, cp)
			case "DeleteMarker", "Version":
				var v Version
				if err = d.DecodeElement(&v, &se); err != nil {
					return err
				}
				if tagName == "DeleteMarker" {
					v.isDeleteMarker = true
				}
				l.Versions = append(l.Versions, v)
			default:
				return errors.New("unrecognized option:" + tagName)
			}
		}
	}
	return nil
}

// ListBucketResult container for listObjects response.
type ListBucketResult struct {
	// A response can contain CommonPrefixes only if you have
	// specified a delimiter.
	CommonPrefixes []CommonPrefix
	// Metadata about each object returned.
	Contents  []ObjectInfo
	Delimiter string

	// Encoding type used to encode object keys in the response.
	EncodingType string

	// A flag that indicates whether or not ListObjects returned all of the results
	// that satisfied the search criteria.
	IsTruncated bool
	Marker      string
	MaxKeys     int64
	Name        string

	// When response is truncated (the IsTruncated element value in
	// the response is true), you can use the key name in this field
	// as marker in the subsequent request to get next set of objects.
	// Object storage lists objects in alphabetical order Note: This
	// element is returned only if you have delimiter request
	// parameter specified. If response does not include the NextMaker
	// and it is truncated, you can use the value of the last Key in
	// the response as the marker in the subsequent request to get the
	// next set of object keys.
	NextMarker string
	Prefix     string
}

// ListMultipartUploadsResult container for ListMultipartUploads response
type ListMultipartUploadsResult struct {
	Bucket             string
	KeyMarker          string
	UploadIDMarker     string `xml:"UploadIdMarker"`
	NextKeyMarker      string
	NextUploadIDMarker string `xml:"NextUploadIdMarker"`
	EncodingType       string
	MaxUploads         int64
	IsTruncated        bool
	Uploads            []ObjectMultipartInfo `xml:"Upload"`
	Prefix             string
	Delimiter          string
	// A response can contain CommonPrefixes only if you specify a delimiter.
	CommonPrefixes []CommonPrefix
}

// initiator container for who initiated multipart upload.
type initiator struct {
	ID          string
	DisplayName string
}

// copyObjectResult container for copy object response.
type copyObjectResult struct {
	ETag         string
	LastModified time.Time // time string format "2006-01-02T15:04:05.000Z"
}

// ObjectPart container for particular part of an object.
type ObjectPart struct {
	// Part number identifies the part.
	PartNumber int

	// Date and time the part was uploaded.
	LastModified time.Time

	// Entity tag returned when the part was uploaded, usually md5sum
	// of the part.
	ETag string

	// Size of the uploaded part data.
	Size int64

	// Checksum values of each part.
	ChecksumCRC32     string
	ChecksumCRC32C    string
	ChecksumSHA1      string
	ChecksumSHA256    string
	ChecksumCRC64NVME string
}

// Checksum will return the checksum for the given type.
// Will return the empty string if not set.
func (c ObjectPart) Checksum(t ChecksumType) string {
	switch {
	case t.Is(ChecksumCRC32C):
		return c.ChecksumCRC32C
	case t.Is(ChecksumCRC32):
		return c.ChecksumCRC32
	case t.Is(ChecksumSHA1):
		return c.ChecksumSHA1
	case t.Is(ChecksumSHA256):
		return c.ChecksumSHA256
	case t.Is(ChecksumCRC64NVME):
		return c.ChecksumCRC64NVME
	}
	return ""
}

// ChecksumRaw returns the decoded checksum from the part.
func (c ObjectPart) ChecksumRaw(t ChecksumType) ([]byte, error) {
	b64 := c.Checksum(t)
	if b64 == "" {
		return nil, errors.New("no checksum set")
	}
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(decoded) != t.RawByteLen() {
		return nil, errors.New("checksum length mismatch")
	}
	return decoded, nil
}

// ListObjectPartsResult container for ListObjectParts response.
type ListObjectPartsResult struct {
	Bucket   string
	Key      string
	UploadID string `xml:"UploadId"`

	Initiator initiator
	Owner     owner

	StorageClass         string
	PartNumberMarker     int
	NextPartNumberMarker int
	MaxParts             int

	// ChecksumAlgorithm will be CRC32, CRC32C, etc.
	ChecksumAlgorithm string

	// ChecksumType is FULL_OBJECT or COMPOSITE (assume COMPOSITE when unset)
	ChecksumType string

	// Indicates whether the returned list of parts is truncated.
	IsTruncated bool
	ObjectParts []ObjectPart `xml:"Part"`

	EncodingType string
}

// initiateMultipartUploadResult container for InitiateMultiPartUpload
// response.
type initiateMultipartUploadResult struct {
	Bucket   string
	Key      string
	UploadID string `xml:"UploadId"`
}

// completeMultipartUploadResult container for completed multipart
// upload response.
type completeMultipartUploadResult struct {
	Location string
	Bucket   string
	Key      string
	ETag     string

	// Checksum values, hash of hashes of parts.
	ChecksumCRC32     string
	ChecksumCRC32C    string
	ChecksumSHA1      string
	ChecksumSHA256    string
	ChecksumCRC64NVME string
	ChecksumType      string
}

// CompletePart sub container lists individual part numbers and their
// md5sum, part of completeMultipartUpload.
type CompletePart struct {
	// Part number identifies the part.
	PartNumber int
	ETag       string

	// Checksum values
	ChecksumCRC32     string `xml:"ChecksumCRC32,omitempty"`
	ChecksumCRC32C    string `xml:"ChecksumCRC32C,omitempty"`
	ChecksumSHA1      string `xml:"ChecksumSHA1,omitempty"`
	ChecksumSHA256    string `xml:"ChecksumSHA256,omitempty"`
	ChecksumCRC64NVME string `xml:",omitempty"`
}

// Checksum will return the checksum for the given type.
// Will return the empty string if not set.
func (c CompletePart) Checksum(t ChecksumType) string {
	switch {
	case t.Is(ChecksumCRC32C):
		return c.ChecksumCRC32C
	case t.Is(ChecksumCRC32):
		return c.ChecksumCRC32
	case t.Is(ChecksumSHA1):
		return c.ChecksumSHA1
	case t.Is(ChecksumSHA256):
		return c.ChecksumSHA256
	case t.Is(ChecksumCRC64NVME):
		return c.ChecksumCRC64NVME
	}
	return ""
}

// completeMultipartUpload container for completing multipart upload.
type completeMultipartUpload struct {
	XMLName xml.Name       `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUpload" json:"-"`
	Parts   []CompletePart `xml:"Part"`
}

// createBucketConfiguration container for bucket configuration.
type createBucketConfiguration struct {
	XMLName  xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CreateBucketConfiguration" json:"-"`
	Location string   `xml:"LocationConstraint"`
}

// deleteObject container for Delete element in MultiObjects Delete XML request
type deleteObject struct {
	Key       string
	VersionID string `xml:"VersionId,omitempty"`
}

// deletedObject container for Deleted element in MultiObjects Delete XML response
type deletedObject struct {
	Key       string
	VersionID string `xml:"VersionId,omitempty"`
	// These fields are ignored.
	DeleteMarker          bool
	DeleteMarkerVersionID string `xml:"DeleteMarkerVersionId,omitempty"`
}

// nonDeletedObject container for Error element (failed deletion) in MultiObjects Delete XML response
type nonDeletedObject struct {
	Key       string
	Code      string
	Message   string
	VersionID string `xml:"VersionId"`
}

// deletedMultiObjects container for MultiObjects Delete XML request
type deleteMultiObjects struct {
	XMLName xml.Name `xml:"Delete"`
	Quiet   bool
	Objects []deleteObject `xml:"Object"`
}

// deletedMultiObjectsResult container for MultiObjects Delete XML response
type deleteMultiObjectsResult struct {
	XMLName          xml.Name           `xml:"DeleteResult"`
	DeletedObjects   []deletedObject    `xml:"Deleted"`
	UnDeletedObjects []nonDeletedObject `xml:"Error"`
}
