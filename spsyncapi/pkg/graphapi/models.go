package graphapi

type Drive struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	CreatedDateTime      string `json:"createdDateTime"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`
	Size                 int64  `json:"size"`
}

type DriveItem struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	FilePath             string `json:"file_path,omitempty"`
	Size                 int64  `json:"size"`
	DownloadUrl          string `json:"@microsoft.graph.downloadUrl"`
	CreatedDateTime      string `json:"createdDateTime"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`

	Folder struct {
		ChildCount int `json:"childCount"`
	} `json:"folder"`
}

func (d *DriveItem) IsFolder() bool {
	return d.Folder.ChildCount > 0
}
