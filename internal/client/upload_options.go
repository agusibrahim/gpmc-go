package client

import "time"

// ProgressStatus represents the current state of a file upload
type ProgressStatus string

const (
	StatusHashing    ProgressStatus = "Hashing"
	StatusUploading  ProgressStatus = "Uploading"
	StatusCommitting ProgressStatus = "Committing"
	StatusDone       ProgressStatus = "Done"
	StatusError      ProgressStatus = "Error"
	StatusSkipped    ProgressStatus = "Skipped"
	StatusBatchMeta  ProgressStatus = "BatchMeta"
)

// ProgressUpdate represents a single progress update for a file
type ProgressUpdate struct {
	ID              string         `json:"id"`
	Filename        string         `json:"filename"`
	Status          ProgressStatus `json:"status"`
	Progress        float64        `json:"progress"`
	Path            string         `json:"path"`
	Error           string         `json:"error,omitempty"`
	MediaKey        string         `json:"media_key,omitempty"`
	TotalFiles      int            `json:"total_files,omitempty"`
	BytesSent       int64          `json:"bytes_sent,omitempty"`
	BytesTotal      int64          `json:"bytes_total,omitempty"`
	BatchBytesSent  int64          `json:"batch_bytes_sent,omitempty"`
	BatchBytesTotal int64          `json:"batch_bytes_total,omitempty"`
	Attempt         int            `json:"attempt,omitempty"`
}

type BatchSummary struct {
	TotalFiles    int           `json:"total_files"`
	Uploaded      int           `json:"uploaded"`
	Skipped       int           `json:"skipped"`
	Failed        int           `json:"failed"`
	TotalBytes    int64         `json:"total_bytes"`
	UploadedBytes int64         `json:"uploaded_bytes"`
	Duration      time.Duration `json:"duration"`
}

type FileResult struct {
	Path     string         `json:"path"`
	Filename string         `json:"filename"`
	Size     int64          `json:"size"`
	ModTime  int64          `json:"mod_time,omitempty"`
	MediaKey string         `json:"media_key,omitempty"`
	Status   ProgressStatus `json:"status"`
	Error    string         `json:"error,omitempty"`
	Hash     string         `json:"hash,omitempty"`
	Verified bool           `json:"verified"`
	Attempts int            `json:"attempts"`
}

type UploadBatchResult struct {
	Summary BatchSummary `json:"summary"`
	Files   []FileResult `json:"files"`
}

// uploadOptions contains options for uploading files
type uploadOptions struct {
	albumName        string
	useQuota         bool
	saver            bool
	recursive        bool
	showProgress     bool
	threads          int
	forceUpload      bool
	deleteFromHost   bool
	filterExp        string
	filterExclude    bool
	filterRegex      bool
	filterIgnoreCase bool
	filterMatchPath  bool
	progressChan     chan ProgressUpdate
	resume           bool
	manifestPath     string
	verify           bool
}

// UploadOption is a functional option for upload operations
type UploadOption func(*uploadOptions)

func WithAlbum(albumName string) UploadOption {
	return func(o *uploadOptions) { o.albumName = albumName }
}

func WithUseQuota(useQuota bool) UploadOption {
	return func(o *uploadOptions) { o.useQuota = useQuota }
}

func WithSaver(saver bool) UploadOption {
	return func(o *uploadOptions) { o.saver = saver }
}

func WithRecursive(recursive bool) UploadOption {
	return func(o *uploadOptions) { o.recursive = recursive }
}

func WithProgress(showProgress bool) UploadOption {
	return func(o *uploadOptions) { o.showProgress = showProgress }
}

func WithThreads(threads int) UploadOption {
	return func(o *uploadOptions) { o.threads = threads }
}

func WithForceUpload(forceUpload bool) UploadOption {
	return func(o *uploadOptions) { o.forceUpload = forceUpload }
}

func WithDeleteFromHost(deleteFromHost bool) UploadOption {
	return func(o *uploadOptions) { o.deleteFromHost = deleteFromHost }
}

func WithFilter(filterExp string) UploadOption {
	return func(o *uploadOptions) { o.filterExp = filterExp }
}

func WithFilterExclude(exclude bool) UploadOption {
	return func(o *uploadOptions) { o.filterExclude = exclude }
}

func WithFilterRegex(useRegex bool) UploadOption {
	return func(o *uploadOptions) { o.filterRegex = useRegex }
}

func WithFilterIgnoreCase(ignoreCase bool) UploadOption {
	return func(o *uploadOptions) { o.filterIgnoreCase = ignoreCase }
}

func WithFilterMatchPath(matchPath bool) UploadOption {
	return func(o *uploadOptions) { o.filterMatchPath = matchPath }
}

func WithProgressChan(ch chan ProgressUpdate) UploadOption {
	return func(o *uploadOptions) { o.progressChan = ch }
}

func WithResume(resume bool) UploadOption {
	return func(o *uploadOptions) { o.resume = resume }
}

func WithManifestPath(path string) UploadOption {
	return func(o *uploadOptions) { o.manifestPath = path }
}

func WithVerify(verify bool) UploadOption {
	return func(o *uploadOptions) { o.verify = verify }
}
