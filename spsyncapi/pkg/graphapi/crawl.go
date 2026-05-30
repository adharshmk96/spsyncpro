package graphapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// FolderCursor identifies a SharePoint folder to enumerate.
type FolderCursor struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

// DriveCrawlState holds resumable folder-walk progress for a single document library.
type DriveCrawlState struct {
	PendingFolders []FolderCursor `json:"pending_folders,omitempty"`
	Current        *FolderCursor  `json:"current,omitempty"`
	PageURL        string         `json:"page_url,omitempty"`
}

// NewDriveCrawlState returns initial crawl state for a drive root.
func NewDriveCrawlState(driveID string) *DriveCrawlState {
	return &DriveCrawlState{
		Current: &FolderCursor{
			URL:  fmt.Sprintf("%s/drives/%s/root/children", graphAPIURL, driveID),
			Path: "",
		},
	}
}

// ListDrivesPage fetches one page of drives for a site. pageURL empty starts at the first page.
func (ms *service) ListDrivesPage(siteID, pageURL string) ([]Drive, string, error) {
	if pageURL == "" {
		pageURL = fmt.Sprintf("%s/sites/%s/drives", graphAPIURL, siteID)
	}

	status, body, err := ms.FetchFromGraphApi(pageURL)
	if err != nil {
		return nil, "", err
	}
	if status != http.StatusOK {
		return nil, "", ErrFailedToFetchData
	}

	var response graphResponse[Drive]
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, "", err
	}
	return response.Value, response.Next, nil
}

// ListDriveItemsPage fetches one Graph page of drive items. state nil starts at drive root.
// done is true when all folders and pages have been processed.
func (ms *service) ListDriveItemsPage(driveID string, state *DriveCrawlState) ([]DriveItem, *DriveCrawlState, bool, error) {
	if state == nil {
		state = NewDriveCrawlState(driveID)
	}

	fetchURL, err := ms.nextDriveItemsFetchURL(driveID, state)
	if err != nil {
		return nil, state, false, err
	}
	if fetchURL == "" {
		return nil, state, true, nil
	}

	status, body, err := ms.FetchFromGraphApi(fetchURL)
	if err != nil {
		return nil, state, false, err
	}
	if status != http.StatusOK {
		return nil, state, false, ErrFailedToFetchData
	}

	var response graphResponse[DriveItem]
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, state, false, err
	}

	currentPath := ""
	if state.Current != nil {
		currentPath = state.Current.Path
	}

	var files []DriveItem
	for _, item := range response.Value {
		fullPath := currentPath + "/" + item.Name
		if item.IsFolder() {
			newURL := fmt.Sprintf("%s/drives/%s/items/%s/children", graphAPIURL, driveID, item.ID)
			state.PendingFolders = append(state.PendingFolders, FolderCursor{URL: newURL, Path: fullPath})
			continue
		}
		item.FilePath = fullPath
		files = append(files, item)
	}

	if response.Next != "" {
		state.PageURL = response.Next
		return files, state, false, nil
	}

	state.PageURL = ""
	state.Current = nil
	return files, state, false, nil
}

func (ms *service) nextDriveItemsFetchURL(driveID string, state *DriveCrawlState) (string, error) {
	if state.PageURL != "" {
		return state.PageURL, nil
	}
	if state.Current != nil {
		return state.Current.URL, nil
	}
	for len(state.PendingFolders) > 0 {
		idx := len(state.PendingFolders) - 1
		state.Current = &state.PendingFolders[idx]
		state.PendingFolders = state.PendingFolders[:idx]
		return state.Current.URL, nil
	}
	_ = driveID
	return "", nil
}
