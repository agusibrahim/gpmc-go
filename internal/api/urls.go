package api

// API endpoint URLs for Google Photos mobile API

const (
	// Base URLs
	BaseAuthURL       = "https://android.googleapis.com/auth"
	BaseUploadURL     = "https://photos.googleapis.com/data/upload/uploadmedia/interactive"
	BasePhotosDataURL = "https://photosdata-pa.googleapis.com/6439526531001121323"
	BaseThumbnailURL  = "https://ap2.googleusercontent.com/gpa"
	BaseStreamURL     = "https://lh3.googleusercontent.com/p"

	// Endpoint IDs (appended to BasePhotosDataURL)
	EndpointCommitUpload       = "/16538846908252377752"
	EndpointFindByHash         = "/5084965799730810217"
	EndpointTrashDelete        = "/17490284929287180316"
	EndpointCreateAlbum        = "/8386163679468898444"
	EndpointAddToAlbum         = "/484917746253879292"
	EndpointLibraryState       = "/18047484249733410717"
	EndpointSetCaption         = "/1552790390512470739"
	EndpointSetFavorite        = "/5144645502632292153"
	EndpointSetArchived        = "/6715446385130606868"

	// Modern Android Album Endpoints discovered via HAR MITM
	EndpointListPhotosInAlbum   = "/2035722585448626696"
	EndpointCommentAlbum        = "/15273438921077897261"
	EndpointAddToAlbumV2        = "/4733640162746355126"
	EndpointRenameAlbum         = "/16466587394238175348"
	EndpointShareAlbum          = "/11663664809460121647"
	EndpointDeleteRegularAlbum  = "/11165707358190966680"
	EndpointDeleteSharedAlbum   = "/8089014401670041416"

	// Full URL for download endpoint (different pattern)
	DownloadURL = "https://photosdata-pa.googleapis.com/$rpc/social.frontend.photos.preparedownloaddata.v1.PhotosPrepareDownloadDataService/PhotosPrepareDownload"

	// AI Enhancement Endpoint (Magic Editor Preset)
	URLMagicEditorPresetEffect = "https://photosdata-pa.googleapis.com/$rpc/social.frontend.photos.effectsdata.v1.PhotosEffectsDataService/PhotosGenerateMagicEditorPresetEffect"
)

// GetUploadURL returns the full upload URL
func GetUploadURL() string {
	return BaseUploadURL
}

// GetUploadURLWithToken returns the upload URL with upload token parameter
func GetUploadURLWithToken(uploadToken string) string {
	return BaseUploadURL + "?upload_id=" + uploadToken
}

// GetPhotosDataURL returns the full photos data URL for a given endpoint
func GetPhotosDataURL(endpoint string) string {
	return BasePhotosDataURL + endpoint
}

// GetThumbnailURL returns the thumbnail URL for a media key
func GetThumbnailURL(mediaKey string) string {
	return BaseThumbnailURL + "/" + mediaKey + "=k-sg"
}

// GetStreamManifestURL returns the stream manifest URL for a media key
func GetStreamManifestURL(mediaKey, protocol string) string {
	return BaseStreamURL + "/" + mediaKey + "%3Dmm%2C" + protocol + "-vm"
}
