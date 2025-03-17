// pkg/storage/file_storage.go

package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

const (
	StatusSuccess = "OK"
	StatusError   = "ERR"
)

// FileInfo represents information about a stored file
type FileInfo struct {
	FileExt      string    `json:"file_ext"`
	FileID       string    `json:"file_id"`
	FileMimeType string    `json:"file_mimetype"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	PublicLink   string    `json:"public_link"`
	Tag          string    `json:"tag"`
	Timestamp    time.Time `json:"timestamp"`
	Bucket       string    `json:"bucket,omitempty"`
}

// FileResponse represents a standard response for file operations
type FileResponse struct {
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	Data       string    `json:"data,omitempty"`
	FileID     string    `json:"file_id,omitempty"`
	Info       *FileInfo `json:"info,omitempty"`
	URL        string    `json:"url,omitempty"`
	ExpiredAt  time.Time `json:"expired_at,omitempty"`
	StringData string    `json:"string_data,omitempty"`
	StreamData io.Reader `json:"-"`
}

// TokenManager handles token operations
type TokenManager interface {
	GenerateToken() (string, error)
	GetToken() (string, error)
	HasToken() bool
}

// FileStorageManager manages file storage operations
type FileStorageManager struct {
	tokenManager TokenManager
	maxRetry     int
	config       *Config
}

// Config holds configuration for file storage
type Config struct {
	HostURI                string
	AuthorizationServerURI string
	ClientID               string
	ClientSecret           string
	AWSKey                 string
	AWSSecret              string
	AWSRegion              string
	AWSBucket              string
	GCSKeyPath             string
	GCSProjectID           string
	GCSBucket              string
}

// NewFileStorageManager creates a new FileStorageManager instance
func NewFileStorageManager(config *Config, tokenManager TokenManager) *FileStorageManager {
	return &FileStorageManager{
		tokenManager: tokenManager,
		maxRetry:     3,
		config:       config,
	}
}

// SetMaxRetry sets the maximum number of retry attempts
func (f *FileStorageManager) SetMaxRetry(maxRetry int) {
	f.maxRetry = maxRetry
}

// UploadBase64File uploads a base64 encoded file
func (f *FileStorageManager) UploadBase64File(filename, extension, mimetype, base64file string) (*FileResponse, error) {
	if filename == "" || extension == "" || mimetype == "" || base64file == "" {
		return nil, fmt.Errorf("invalid arguments")
	}

	reqBody := map[string]string{
		"file_name":       filename,
		"file_ext":        extension,
		"mime_type":       mimetype,
		"binary_data_b64": base64file,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	attempts := 0
	var resp *http.Response

	for {
		if attempts >= f.maxRetry {
			return nil, fmt.Errorf("exceeded maximum retry attempts")
		}

		token, err := f.tokenManager.GetToken()
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", f.config.HostURI+"/d/files", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-code", token)
		req.Header.Set("x-client-id", f.config.ClientID)

		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			attempts++
			f.tokenManager.GenerateToken()
			continue
		}

		if resp.StatusCode >= 500 {
			attempts++
			f.tokenManager.GenerateToken()
			resp.Body.Close()
			continue
		}

		break
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fileResponse FileResponse
	err = json.Unmarshal(body, &fileResponse)
	if err != nil {
		return nil, err
	}

	return &fileResponse, nil
}

// Upload uploads a file
func (f *FileStorageManager) Upload(file *multipart.FileHeader) (*FileResponse, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	data, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}

	// Get filename and extension
	filename := filepath.Base(file.Filename)
	extension := filepath.Ext(filename)
	if extension != "" {
		extension = extension[1:] // Remove the dot
	}

	// Clean filename
	filename = filename[:len(filename)-len(extension)-1]

	// Encode file data as base64
	base64Data := base64.StdEncoding.EncodeToString(data)

	return f.UploadBase64File(filename, extension, file.Header.Get("Content-Type"), base64Data)
}

// Delete deletes a file by ID
func (f *FileStorageManager) Delete(fileID string) (*FileResponse, error) {
	attempts := 0
	var resp *http.Response

	for {
		if attempts >= f.maxRetry {
			return nil, fmt.Errorf("exceeded maximum retry attempts")
		}

		token, err := f.tokenManager.GetToken()
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("DELETE", f.config.HostURI+"/d/files/"+fileID, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-code", token)
		req.Header.Set("x-client-id", f.config.ClientID)

		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			attempts++
			f.tokenManager.GenerateToken()
			continue
		}

		if resp.StatusCode >= 500 {
			attempts++
			f.tokenManager.GenerateToken()
			resp.Body.Close()
			continue
		}

		break
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fileResponse FileResponse
	err = json.Unmarshal(body, &fileResponse)
	if err != nil {
		return nil, err
	}

	return &fileResponse, nil
}

// GetFileById retrieves file information by ID
func (f *FileStorageManager) GetFileById(fileID string) (*FileResponse, error) {
	attempts := 0
	var resp *http.Response

	for {
		if attempts >= f.maxRetry {
			return nil, fmt.Errorf("exceeded maximum retry attempts")
		}

		token, err := f.tokenManager.GetToken()
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("GET", f.config.HostURI+"/d/files/"+fileID, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-code", token)
		req.Header.Set("x-client-id", f.config.ClientID)

		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			attempts++
			f.tokenManager.GenerateToken()
			continue
		}

		if resp.StatusCode >= 500 {
			attempts++
			f.tokenManager.GenerateToken()
			resp.Body.Close()
			continue
		}

		break
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fileResponse FileResponse
	err = json.Unmarshal(body, &fileResponse)
	if err != nil {
		return nil, err
	}

	return &fileResponse, nil
}

// GetAwsClient returns an AWS S3 client
func (f *FileStorageManager) GetAwsClient() (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(f.config.AWSRegion),
		Credentials: credentials.NewStaticCredentials(f.config.AWSKey, f.config.AWSSecret, ""),
	})
	if err != nil {
		return nil, err
	}

	return s3.New(sess), nil
}

// AwsUpload uploads a file to AWS S3
func (f *FileStorageManager) AwsUpload(file *multipart.FileHeader, subdirectory string, bucketname string) (*FileResponse, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	data, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}

	// Get filename and extension
	origFilename := filepath.Base(file.Filename)
	extension := filepath.Ext(origFilename)
	if extension != "" {
		extension = extension[1:] // Remove the dot
	}

	// Generate a unique filename
	uniqueFilename := uuid.New().String() + "." + extension

	// Add subdirectory if provided
	var fileID string
	if subdirectory != "" {
		fileID = subdirectory + "/" + uniqueFilename
	} else {
		fileID = uniqueFilename
	}

	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.AWSBucket
	}

	// Get AWS S3 client
	s3Client, err := f.GetAwsClient()
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Upload to S3
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(bucketname),
		Key:           aws.String(fileID),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String(file.Header.Get("Content-Type")),
	})

	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Generate public URL
	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketname, f.config.AWSRegion, fileID)

	// Create response
	fileInfo := &FileInfo{
		FileExt:      extension,
		FileID:       fileID,
		FileMimeType: file.Header.Get("Content-Type"),
		FileName:     origFilename[:len(origFilename)-len(extension)-1],
		FileSize:     int64(len(data)),
		PublicLink:   publicURL,
		Tag:          "", // ETag not available without GetObjectOutput
		Timestamp:    time.Now(),
	}

	response := &FileResponse{
		Status:  StatusSuccess,
		Message: "INSERT " + fileID,
		FileID:  fileID,
		Info:    fileInfo,
	}

	return response, nil
}

// AwsDelete deletes a file from AWS S3
func (f *FileStorageManager) AwsDelete(awsFileID string, bucketname string) (*FileResponse, error) {
	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.AWSBucket
	}

	// Get AWS S3 client
	s3Client, err := f.GetAwsClient()
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Delete from S3
	_, err = s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketname),
		Key:    aws.String(awsFileID),
	})

	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create response
	response := &FileResponse{
		Status:  StatusSuccess,
		Message: "DELETE " + awsFileID,
	}

	return response, nil
}

// AwsGetFileById retrieves file information from AWS S3
func (f *FileStorageManager) AwsGetFileById(awsFileID string, bucketname string) (*FileResponse, error) {
	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.AWSBucket
	}

	// Get AWS S3 client
	s3Client, err := f.GetAwsClient()
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Get from S3
	result, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketname),
		Key:    aws.String(awsFileID),
	})

	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Read the file data
	body, err := ioutil.ReadAll(result.Body)
	result.Body.Close()
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Get file information
	extension := filepath.Ext(awsFileID)
	if extension != "" {
		extension = extension[1:] // Remove the dot
	}

	// Generate public URL
	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketname, f.config.AWSRegion, awsFileID)

	// Create response
	fileInfo := &FileInfo{
		FileExt:      extension,
		FileID:       awsFileID,
		FileMimeType: aws.StringValue(result.ContentType),
		FileSize:     aws.Int64Value(result.ContentLength),
		PublicLink:   publicURL,
		Tag:          aws.StringValue(result.ETag),
		Timestamp:    time.Now(),
	}

	// Create response
	response := &FileResponse{
		Status: StatusSuccess,
		Data:   base64.StdEncoding.EncodeToString(body),
		Info:   fileInfo,
	}

	return response, nil
}

func (f *FileStorageManager) AwsDownloadFile(awsFileID string, bucketname string, saveAsPath string) (*FileResponse, error) {
	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.AWSBucket
	}

	// Get AWS S3 client
	s3Client, err := f.GetAwsClient()
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create the file
	file, err := os.Create(saveAsPath)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer file.Close()

	// Download from S3
	result, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketname),
		Key:    aws.String(awsFileID),
	})

	if err != nil {
		// Remove file if it was created
		os.Remove(saveAsPath)
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Copy to file
	_, err = io.Copy(file, result.Body)
	result.Body.Close()
	if err != nil {
		// Remove file if it was created
		os.Remove(saveAsPath)
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create response
	response := &FileResponse{
		Status:  StatusSuccess,
		Message: "File success saved to " + saveAsPath,
	}

	return response, nil
}

// AwsGetTemporaryPublicLink generates a temporary public URL for an AWS S3 file
func (f *FileStorageManager) AwsGetTemporaryPublicLink(awsFileID string, expiry time.Time, bucketname string) (*FileResponse, error) {
	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.AWSBucket
	}

	// Set default expiry if not specified
	if expiry.IsZero() {
		expiry = time.Now().Add(30 * time.Minute)
	}

	// Get AWS S3 client
	s3Client, err := f.GetAwsClient()
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create request for pre-signed URL
	req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketname),
		Key:    aws.String(awsFileID),
	})

	// Generate pre-signed URL
	urlStr, err := req.Presign(expiry.Sub(time.Now()))
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create response
	response := &FileResponse{
		Status:    StatusSuccess,
		URL:       urlStr,
		ExpiredAt: expiry,
	}

	return response, nil
}

// GetGcsClient returns a Google Cloud Storage client
func (f *FileStorageManager) GetGcsClient(projectID string) (*storage.Client, error) {
	ctx := context.Background()

	// Use default project ID if not specified
	if projectID == "" {
		projectID = f.config.GCSProjectID
	}

	// Check if key path and project ID are provided
	if f.config.GCSKeyPath == "" || projectID == "" {
		return nil, fmt.Errorf("credential not found")
	}

	// Create GCS client
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(f.config.GCSKeyPath))
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GcsUpload uploads a file to Google Cloud Storage
func (f *FileStorageManager) GcsUpload(file *multipart.FileHeader, subdirectory string, bucketname string, projectID string) (*FileResponse, error) {
	ctx := context.Background()

	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	data, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}

	// Get filename and extension
	origFilename := filepath.Base(file.Filename)
	extension := filepath.Ext(origFilename)
	if extension != "" {
		extension = extension[1:] // Remove the dot
	}

	// Generate a unique filename
	uniqueFilename := uuid.New().String() + "." + extension

	// Add subdirectory if provided
	var fileID string
	if subdirectory != "" {
		fileID = subdirectory + "/" + uniqueFilename
	} else {
		fileID = uniqueFilename
	}

	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.GCSBucket
	}

	// Get GCS client
	gcsClient, err := f.GetGcsClient(projectID)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer gcsClient.Close()

	// Get bucket handle
	bucket := gcsClient.Bucket(bucketname)

	// Check if bucket exists
	_, err = bucket.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Bucket not found",
		}, nil
	}

	// Create object handle
	obj := bucket.Object(fileID)

	// Upload data
	wc := obj.NewWriter(ctx)
	wc.ContentType = file.Header.Get("Content-Type")

	if _, err := wc.Write(data); err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	if err := wc.Close(); err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Get object attributes
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Generate public URL
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketname, fileID)

	// Create response
	fileInfo := &FileInfo{
		FileExt:      extension,
		FileID:       fileID,
		FileMimeType: attrs.ContentType,
		FileName:     origFilename[:len(origFilename)-len(extension)-1],
		FileSize:     attrs.Size,
		PublicLink:   publicURL,
		Tag:          attrs.Etag,
		Timestamp:    attrs.Created,
		Bucket:       bucketname,
	}

	response := &FileResponse{
		Status:  StatusSuccess,
		Message: "INSERT " + fileID,
		FileID:  fileID,
		Info:    fileInfo,
	}

	return response, nil
}

// GcsDelete deletes a file from Google Cloud Storage
func (f *FileStorageManager) GcsDelete(gcsFileID string, bucketname string, projectID string) (*FileResponse, error) {
	ctx := context.Background()

	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.GCSBucket
	}

	// Get GCS client
	gcsClient, err := f.GetGcsClient(projectID)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer gcsClient.Close()

	// Get bucket handle
	bucket := gcsClient.Bucket(bucketname)

	// Check if bucket exists
	_, err = bucket.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Bucket not found",
		}, nil
	}

	// Create object handle
	obj := bucket.Object(gcsFileID)

	// Check if object exists
	_, err = obj.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Object " + gcsFileID + " not found",
		}, nil
	}

	// Delete object
	if err := obj.Delete(ctx); err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create response
	response := &FileResponse{
		Status:  StatusSuccess,
		Message: "DELETE " + gcsFileID,
	}

	return response, nil
}

// GcsGetFileById retrieves file information from Google Cloud Storage
func (f *FileStorageManager) GcsGetFileById(gcsFileID string, bucketname string, projectID string) (*FileResponse, error) {
	ctx := context.Background()

	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.GCSBucket
	}

	// Get GCS client
	gcsClient, err := f.GetGcsClient(projectID)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer gcsClient.Close()

	// Get bucket handle
	bucket := gcsClient.Bucket(bucketname)

	// Check if bucket exists
	_, err = bucket.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Bucket not found",
		}, nil
	}

	// Create object handle
	obj := bucket.Object(gcsFileID)

	// Check if object exists
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Object " + gcsFileID + " not found",
		}, nil
	}

	// Read the file data
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Get file information
	extension := filepath.Ext(gcsFileID)
	if extension != "" {
		extension = extension[1:] // Remove the dot
	}

	// Generate public URL
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketname, gcsFileID)

	// Create response
	fileInfo := &FileInfo{
		FileExt:      extension,
		FileID:       gcsFileID,
		FileMimeType: attrs.ContentType,
		FileSize:     attrs.Size,
		PublicLink:   publicURL,
		Tag:          attrs.Etag,
		Timestamp:    attrs.Created,
		Bucket:       bucketname,
	}

	response := &FileResponse{
		Status: StatusSuccess,
		Data:   base64.StdEncoding.EncodeToString(data),
		Info:   fileInfo,
	}

	return response, nil
}

// GcsDownloadFile downloads a file from Google Cloud Storage to a local path
func (f *FileStorageManager) GcsDownloadFile(gcsFileID string, saveAsPath string, bucketname string, projectID string) (*FileResponse, error) {
	ctx := context.Background()

	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.GCSBucket
	}

	// Get GCS client
	gcsClient, err := f.GetGcsClient(projectID)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer gcsClient.Close()

	// Get bucket handle
	bucket := gcsClient.Bucket(bucketname)

	// Check if bucket exists
	_, err = bucket.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Bucket not found",
		}, nil
	}

	// Create object handle
	obj := bucket.Object(gcsFileID)

	// Check if object exists
	_, err = obj.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Object " + gcsFileID + " not found",
		}, nil
	}

	// Create the file
	file, err := os.Create(saveAsPath)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer file.Close()

	// Get reader
	reader, err := obj.NewReader(ctx)
	if err != nil {
		os.Remove(saveAsPath)
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer reader.Close()

	// Copy to file
	_, err = io.Copy(file, reader)
	if err != nil {
		os.Remove(saveAsPath)
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create response
	response := &FileResponse{
		Status:  StatusSuccess,
		Message: "File success saved to " + saveAsPath,
	}

	return response, nil
}

// GcsGetFileByIdAsString retrieves file content as a string from Google Cloud Storage
func (f *FileStorageManager) GcsGetFileByIdAsString(gcsFileID string, bucketname string, projectID string) (*FileResponse, error) {
	ctx := context.Background()

	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.GCSBucket
	}

	// Get GCS client
	gcsClient, err := f.GetGcsClient(projectID)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer gcsClient.Close()

	// Get bucket handle
	bucket := gcsClient.Bucket(bucketname)

	// Check if bucket exists
	_, err = bucket.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Bucket not found",
		}, nil
	}

	// Create object handle
	obj := bucket.Object(gcsFileID)

	// Check if object exists
	_, err = obj.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Object " + gcsFileID + " not found",
		}, nil
	}

	// Read the file data
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create response
	response := &FileResponse{
		Status:     StatusSuccess,
		StringData: string(data),
	}

	return response, nil
}

// GcsGetFileByIdAsStream retrieves file content as a stream from Google Cloud Storage
func (f *FileStorageManager) GcsGetFileByIdAsStream(gcsFileID string, bucketname string, projectID string) (*FileResponse, error) {
	ctx := context.Background()

	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.GCSBucket
	}

	// Get GCS client
	gcsClient, err := f.GetGcsClient(projectID)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Get bucket handle
	bucket := gcsClient.Bucket(bucketname)

	// Check if bucket exists
	_, err = bucket.Attrs(ctx)
	if err != nil {
		gcsClient.Close()
		return &FileResponse{
			Status:  StatusError,
			Message: "Bucket not found",
		}, nil
	}

	// Create object handle
	obj := bucket.Object(gcsFileID)

	// Check if object exists
	_, err = obj.Attrs(ctx)
	if err != nil {
		gcsClient.Close()
		return &FileResponse{
			Status:  StatusError,
			Message: "Object " + gcsFileID + " not found",
		}, nil
	}

	// Get reader
	reader, err := obj.NewReader(ctx)
	if err != nil {
		gcsClient.Close()
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create response with stream (caller must close both the reader and client)
	response := &FileResponse{
		Status:     StatusSuccess,
		StreamData: reader,
	}

	return response, nil
}

// GcsGetTemporaryPublicLink generates a temporary public URL for a GCS file
// In your storage/file_storage.go file, update the GcsGetTemporaryPublicLink function:

func (f *FileStorageManager) GcsGetTemporaryPublicLink(gcsFileID string, expiry time.Time, bucketname string, projectID string) (*FileResponse, error) {
	ctx := context.Background()

	// Use default bucket if not specified
	if bucketname == "" {
		bucketname = f.config.GCSBucket
	}

	// Set default expiry if not specified
	if expiry.IsZero() {
		expiry = time.Now().Add(30 * time.Minute)
	}

	// Get GCS client
	gcsClient, err := f.GetGcsClient(projectID)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}
	defer gcsClient.Close()

	// Get bucket handle
	bucket := gcsClient.Bucket(bucketname)

	// Check if bucket exists
	_, err = bucket.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Bucket not found",
		}, nil
	}

	// Create object handle
	obj := bucket.Object(gcsFileID)

	// Check if object exists
	_, err = obj.Attrs(ctx)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Object " + gcsFileID + " not found",
		}, nil
	}

	// Load the service account key file to get the credentials
	jsonKey, err := ioutil.ReadFile(f.config.GCSKeyPath)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Failed to read service account key: " + err.Error(),
		}, nil
	}

	// Parse the service account key
	var keyData struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}

	if err := json.Unmarshal(jsonKey, &keyData); err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: "Failed to parse service account key: " + err.Error(),
		}, nil
	}

	// Create signed URL options
	opts := &storage.SignedURLOptions{
		Method:         "GET",
		Expires:        expiry,
		GoogleAccessID: keyData.ClientEmail,        // Use the service account email
		PrivateKey:     []byte(keyData.PrivateKey), // Use the private key
	}

	// Generate signed URL
	url, err := storage.SignedURL(bucketname, gcsFileID, opts)
	if err != nil {
		return &FileResponse{
			Status:  StatusError,
			Message: err.Error(),
		}, nil
	}

	// Create response
	response := &FileResponse{
		Status:    StatusSuccess,
		URL:       url,
		ExpiredAt: expiry,
	}

	return response, nil
}
