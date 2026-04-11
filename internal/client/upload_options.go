package client

// uploadOptions contains options for uploading files
type uploadOptions struct {
	albumName          string
	useQuota           bool
	saver              bool
	recursive          bool
	showProgress       bool
	threads            int
	forceUpload        bool
	deleteFromHost     bool
	filterExp          string
	filterExclude      bool
	filterRegex        bool
	filterIgnoreCase   bool
	filterMatchPath    bool
}

// UploadOption is a functional option for upload operations
type UploadOption func(*uploadOptions)

// WithAlbum sets the album name for uploaded files
func WithAlbum(albumName string) UploadOption {
	return func(o *uploadOptions) {
		o.albumName = albumName
	}
}

// WithUseQuota sets whether to count uploads against storage quota
func WithUseQuota(useQuota bool) UploadOption {
	return func(o *uploadOptions) {
		o.useQuota = useQuota
	}
}

// WithSaver sets whether to upload in storage saver quality
func WithSaver(saver bool) UploadOption {
	return func(o *uploadOptions) {
		o.saver = saver
	}
}

// WithRecursive sets whether to scan directories recursively
func WithRecursive(recursive bool) UploadOption {
	return func(o *uploadOptions) {
		o.recursive = recursive
	}
}

// WithProgress sets whether to show upload progress
func WithProgress(showProgress bool) UploadOption {
	return func(o *uploadOptions) {
		o.showProgress = showProgress
	}
}

// WithThreads sets the number of concurrent upload threads
func WithThreads(threads int) UploadOption {
	return func(o *uploadOptions) {
		o.threads = threads
	}
}

// WithForceUpload sets whether to skip hash checking and upload all files
func WithForceUpload(forceUpload bool) UploadOption {
	return func(o *uploadOptions) {
		o.forceUpload = forceUpload
	}
}

// WithDeleteFromHost sets whether to delete files after successful upload
func WithDeleteFromHost(deleteFromHost bool) UploadOption {
	return func(o *uploadOptions) {
		o.deleteFromHost = deleteFromHost
	}
}

// WithFilter sets a filter expression for file selection
func WithFilter(filterExp string) UploadOption {
	return func(o *uploadOptions) {
		o.filterExp = filterExp
	}
}

// WithFilterExclude sets whether to exclude matching files
func WithFilterExclude(exclude bool) UploadOption {
	return func(o *uploadOptions) {
		o.filterExclude = exclude
	}
}

// WithFilterRegex sets whether to use regex for filtering
func WithFilterRegex(useRegex bool) UploadOption {
	return func(o *uploadOptions) {
		o.filterRegex = useRegex
	}
}

// WithFilterIgnoreCase sets whether to use case-insensitive filtering
func WithFilterIgnoreCase(ignoreCase bool) UploadOption {
	return func(o *uploadOptions) {
		o.filterIgnoreCase = ignoreCase
	}
}

// WithFilterMatchPath sets whether to match against full path instead of filename
func WithFilterMatchPath(matchPath bool) UploadOption {
	return func(o *uploadOptions) {
		o.filterMatchPath = matchPath
	}
}
