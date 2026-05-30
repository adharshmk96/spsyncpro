package graphapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"time"
)

const graphAPIURL = "https://graph.microsoft.com/v1.0"
const maxGraphAPIRetry = 3

var (
	ErrAccessTokenNotFound      = errors.New("access_token not found in response")
	ErrFailedToFetchData        = errors.New("failed to fetch data from Graph API")
	ErrDriveNotFound            = errors.New("drive not found")
	ErrDriveIdOrItemPathIsEmpty = errors.New("driveId or itemPath is empty")
	ErrFailedToUploadFile       = errors.New("failed to upload file")
	ErrRetryLimitExceeded       = errors.New("retry limit exceeded")
)

// OrgConfig holds Microsoft Entra credentials for Graph API access.
type OrgConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

// Config wires dependencies for the Graph API client.
type Config struct {
	Org        OrgConfig
	Logger     *slog.Logger
	HTTPClient *http.Client
}

// Service defines SharePoint / Microsoft Graph operations used for backup and restore.
type Service interface {
	GetAccessToken() (string, error)
	ValidateToken(accessToken string) (bool, error)
	FetchFromGraphApi(url string) (int, []byte, error)

	GetSiteId(siteUrl string) (string, error)
	GetDriveId(siteId, driveName string) (string, error)
	GetDriveList(siteId string) <-chan Drive
	GetDriveItems(driveId string) <-chan DriveItem
	GetDriveItemDownload(driveId, itemId string) (*http.Response, error)

	CreateDocumentLibrary(siteId, driveName string) (string, error)
	UploadDriveItemWhole(driveId, itemPath string, reader io.Reader) error
	UploadDriveItemChunked(driveId, itemPath string, totalBytes int64, reader io.Reader) error
}

type service struct {
	orgConfig   OrgConfig
	httpClient  *http.Client
	accessToken string
	logger      *slog.Logger
}

// NewService constructs a Graph API client.
func NewService(cfg Config) Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With(slog.String("module", "graphapi"))

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	return &service{
		orgConfig:  cfg.Org,
		httpClient: client,
		logger:     logger,
	}
}

func (ms *service) GetAccessToken() (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", ms.orgConfig.TenantID)

	data := url.Values{}
	data.Set("client_id", ms.orgConfig.ClientID)
	data.Set("scope", "https://graph.microsoft.com/.default")
	data.Set("client_secret", ms.orgConfig.ClientSecret)
	data.Set("grant_type", "client_credentials")

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", ErrAccessTokenNotFound
	}

	ms.accessToken = result.AccessToken
	return ms.accessToken, nil
}

func (ms *service) ValidateToken(accessToken string) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, graphAPIURL+"/sites", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := ms.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return res.StatusCode == http.StatusOK, nil
}

func (ms *service) GetResponse(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	for i := 0; i < maxGraphAPIRetry; i++ {
		req.Header.Set("Authorization", "Bearer "+ms.accessToken)
		res, err := ms.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if res.StatusCode == http.StatusUnauthorized {
			res.Body.Close()
			if _, err := ms.GetAccessToken(); err != nil {
				return nil, err
			}
			time.Sleep(time.Second)
			continue
		}
		return res, nil
	}

	return nil, ErrRetryLimitExceeded
}

func (ms *service) FetchFromGraphApi(url string) (int, []byte, error) {
	res, err := ms.GetResponse(url)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}
	return res.StatusCode, body, nil
}

func transformURL(siteURL string) (string, error) {
	parsed, err := url.Parse(siteURL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", parsed.Hostname(), parsed.Path), nil
}

func (ms *service) GetSiteId(siteURL string) (string, error) {
	parsed, err := transformURL(siteURL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("%s/sites/%s", graphAPIURL, parsed)
	status, body, err := ms.FetchFromGraphApi(apiURL)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", ErrFailedToFetchData
	}

	var site struct {
		ID string `json:"id"`
	}
	if err = json.Unmarshal(body, &site); err != nil {
		return "", err
	}
	return site.ID, nil
}

type graphResponse[T any] struct {
	Value []T    `json:"value"`
	Next  string `json:"@odata.nextLink"`
}

func (ms *service) GetDriveList(siteID string) <-chan Drive {
	driveChan := make(chan Drive)

	go func() {
		defer close(driveChan)

		driveURL := fmt.Sprintf("%s/sites/%s/drives", graphAPIURL, siteID)
		for driveURL != "" {
			status, body, err := ms.FetchFromGraphApi(driveURL)
			if err != nil {
				ms.logger.Error("failed to fetch drive list", slog.String("error", err.Error()))
				return
			}
			if status != http.StatusOK {
				return
			}

			var response graphResponse[Drive]
			if err = json.Unmarshal(body, &response); err != nil {
				ms.logger.Error("failed to unmarshal drive list", slog.String("error", err.Error()))
				return
			}

			for _, drive := range response.Value {
				driveChan <- drive
			}
			driveURL = response.Next
		}
	}()

	return driveChan
}

func (ms *service) GetDriveId(siteID, driveName string) (string, error) {
	for drive := range ms.GetDriveList(siteID) {
		if drive.Name == driveName {
			return drive.ID, nil
		}
	}
	return "", ErrDriveNotFound
}

func (ms *service) GetDriveItems(driveID string) <-chan DriveItem {
	driveItemChan := make(chan DriveItem)

	go func() {
		defer close(driveItemChan)

		type folder struct {
			url  string
			path string
		}
		stack := []folder{{url: fmt.Sprintf("%s/drives/%s/root/children", graphAPIURL, driveID), path: ""}}

		for len(stack) > 0 {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			driveItemsURL := current.url
			currentPath := current.path

			for driveItemsURL != "" {
				status, body, err := ms.FetchFromGraphApi(driveItemsURL)
				if err != nil {
					ms.logger.Error("failed to fetch drive items", slog.String("error", err.Error()))
					return
				}
				if status != http.StatusOK {
					return
				}

				var response graphResponse[DriveItem]
				if err = json.Unmarshal(body, &response); err != nil {
					ms.logger.Error("failed to unmarshal drive items", slog.String("error", err.Error()))
					return
				}

				for _, item := range response.Value {
					fullPath := currentPath + "/" + item.Name
					if item.IsFolder() {
						newURL := fmt.Sprintf("%s/drives/%s/items/%s/children", graphAPIURL, driveID, item.ID)
						stack = append(stack, folder{url: newURL, path: fullPath})
					} else {
						item.FilePath = fullPath
						driveItemChan <- item
					}
				}
				driveItemsURL = response.Next
			}
		}
	}()

	return driveItemChan
}

func (ms *service) GetDriveItemDownload(driveID, itemID string) (*http.Response, error) {
	downloadURL := fmt.Sprintf("%s/drives/%s/items/%s/content", graphAPIURL, driveID, itemID)
	return ms.GetResponse(downloadURL)
}

func (ms *service) CreateDocumentLibrary(siteID, driveName string) (string, error) {
	driveURL := fmt.Sprintf("%s/sites/%s/lists", graphAPIURL, siteID)

	payload, err := json.Marshal(map[string]interface{}{
		"displayName": driveName,
		"list": map[string]string{
			"template": "documentLibrary",
		},
	})
	if err != nil {
		return "", err
	}

	var res *http.Response
	for i := 0; i < maxGraphAPIRetry; i++ {
		req, err := http.NewRequest(http.MethodPost, driveURL, bytes.NewReader(payload))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+ms.accessToken)
		req.Header.Set("Content-Type", "application/json")

		res, err = ms.httpClient.Do(req)
		if err != nil {
			return "", err
		}

		if res.StatusCode == http.StatusUnauthorized {
			res.Body.Close()
			if _, err := ms.GetAccessToken(); err != nil {
				return "", err
			}
			time.Sleep(time.Second)
			continue
		}
		break
	}
	defer res.Body.Close()

	var response struct {
		ID string `json:"id"`
	}
	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", err
	}
	return response.ID, nil
}

func (ms *service) UploadDriveItemWhole(driveID, itemPath string, reader io.Reader) error {
	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read upload body: %w", err)
	}

	uploadURL := fmt.Sprintf("%s/drives/%s/root:/%s:/content", graphAPIURL, driveID, itemPath)

	var uploadResp *http.Response
	for i := 0; i < maxGraphAPIRetry; i++ {
		req, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+ms.accessToken)
		req.Header.Set("Content-Type", "application/octet-stream")

		uploadResp, err = ms.httpClient.Do(req)
		if err != nil {
			ms.logger.Error("failed to upload file", slog.String("error", err.Error()))
			continue
		}

		if uploadResp.StatusCode == http.StatusUnauthorized {
			uploadResp.Body.Close()
			if _, err := ms.GetAccessToken(); err != nil {
				return err
			}
			time.Sleep(time.Second)
			continue
		}

		if uploadResp.StatusCode == http.StatusOK ||
			uploadResp.StatusCode == http.StatusCreated ||
			uploadResp.StatusCode == http.StatusAccepted {
			uploadResp.Body.Close()
			return nil
		}

		uploadResp.Body.Close()
	}

	if uploadResp != nil {
		ms.logger.Error("failed to upload file after retries", slog.String("status", uploadResp.Status))
	}
	return ErrFailedToUploadFile
}

func (ms *service) UploadDriveItemChunked(driveID, itemPath string, totalBytes int64, reader io.Reader) error {
	uploadSessionURL := fmt.Sprintf("%s/drives/%s/root:/%s:/createUploadSession", graphAPIURL, driveID, itemPath)
	uploadSessionData, err := json.Marshal(map[string]interface{}{
		"item": map[string]string{
			"@microsoft.graph.conflictBehavior": "replace",
			"name":                              filepath.Base(itemPath),
		},
	})
	if err != nil {
		return err
	}

	var uploadSession struct {
		UploadURL string `json:"uploadUrl"`
	}

	for i := 0; i < maxGraphAPIRetry; i++ {
		req, err := http.NewRequest(http.MethodPost, uploadSessionURL, bytes.NewReader(uploadSessionData))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+ms.accessToken)
		req.Header.Set("Content-Type", "application/json")

		createSessionResp, err := ms.httpClient.Do(req)
		if err != nil {
			ms.logger.Error("failed to create upload session", slog.String("error", err.Error()))
			continue
		}

		if createSessionResp.StatusCode == http.StatusUnauthorized {
			createSessionResp.Body.Close()
			if _, err := ms.GetAccessToken(); err != nil {
				return err
			}
			time.Sleep(time.Second)
			continue
		}

		body, err := io.ReadAll(createSessionResp.Body)
		createSessionResp.Body.Close()
		if err != nil {
			return err
		}

		if err = json.Unmarshal(body, &uploadSession); err != nil {
			return err
		}

		if createSessionResp.StatusCode == http.StatusOK || createSessionResp.StatusCode == http.StatusCreated {
			break
		}
	}

	if uploadSession.UploadURL == "" {
		return fmt.Errorf("failed to create upload session after retries")
	}

	const chunkSize = int64(4 * 1024 * 1024)
	buffer := make([]byte, chunkSize)
	startByte := int64(0)

	for {
		bytesRead := 0
		for bytesRead < int(chunkSize) {
			n, err := reader.Read(buffer[bytesRead:])
			bytesRead += n
			if err == io.EOF {
				if bytesRead == 0 {
					return nil
				}
				break
			}
			if err != nil {
				return err
			}
			if n == 0 {
				break
			}
		}

		n := int64(bytesRead)
		endByte := startByte + n - 1

		for i := 0; i < maxGraphAPIRetry; i++ {
			fileUploadReq, err := http.NewRequest(http.MethodPut, uploadSession.UploadURL, bytes.NewReader(buffer[:n]))
			if err != nil {
				return err
			}
			fileUploadReq.Header.Set("Content-Length", fmt.Sprintf("%d", n))
			fileUploadReq.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startByte, endByte, totalBytes))
			fileUploadReq.Header.Set("Content-Type", "application/octet-stream")
			fileUploadReq.Header.Set("Authorization", "Bearer "+ms.accessToken)

			fileUploadResp, err := ms.httpClient.Do(fileUploadReq)
			if err != nil {
				ms.logger.Error("failed to upload chunk", slog.String("error", err.Error()))
				continue
			}

			if fileUploadResp.StatusCode == http.StatusUnauthorized {
				fileUploadResp.Body.Close()
				if _, err := ms.GetAccessToken(); err != nil {
					return err
				}
				time.Sleep(time.Second)
				continue
			}

			fileUploadResp.Body.Close()
			if fileUploadResp.StatusCode != http.StatusOK &&
				fileUploadResp.StatusCode != http.StatusCreated &&
				fileUploadResp.StatusCode != http.StatusAccepted {
				continue
			}
			break
		}

		startByte = endByte + 1
		if startByte >= totalBytes {
			break
		}
	}

	return nil
}
