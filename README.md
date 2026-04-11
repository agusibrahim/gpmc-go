# GPMC - Google Photos Mobile Client

[![Go Version](https://img.shields.io/github/go-mod/go-version/xob0t/gpmc-go)](https://go.dev/)
[![License](https://img.shields.io/github/license/xob0t/gpmc-go)](LICENSE)

GPMC is a high-performance Google Photos client written in Go, based on a reverse-engineered mobile API. It allows for advanced media management and uploading features that are typically restricted in the standard web API.

## 🚀 Features

- **Unlimited Original Quality Uploads**: Leverage mobile API signatures to upload media in original quality (mimicking mobile device behavior).
- **Interactive Web Dashboard**: A premium, "Shadcn-inspired" web interface with dark mode support and real-time upload tracking.
- **Concurrent Uploads**: Multi-threaded upload system for maximum throughput.
- **Smart Deduplication**: Hash-based detection to prevent uploading files already present in your Google Photos library.
- **Album Management**: Auto-create albums based on directory structure or specify custom target albums.
- **Local Library Cache**: Incrementally sync your remote library metadata to a local SQLite database for fast lookups.
- **Advanced Filtering**: Support for regex, recursive scanning, and file exclusion filters.

## 🛠 Installation

### From Source
Ensure you have Go 1.25 or later installed.

```bash
git clone https://github.com/xob0t/gpmc-go.git
cd gpmc-go
go build -o gpmc ./cmd/gpmc
```

## ⚙️ Configuration

GPMC requires `auth_data` to authenticate with Google Services. This is typically obtained by capturing a request from the Google Photos mobile app.

1. Initialize the default configuration:
   ```bash
   ./gpmc init-config
   ```
2. Edit `~/.gpmc/config.yaml` and provide your `auth_data`:
   ```yaml
   auth_data: "YOUR_URL_ENCODED_AUTH_DATA"
   threads: 8
   log_level: "INFO"
   ```

Alternatively, you can set the `GP_AUTH_DATA` environment variable.

## 📖 Usage

### CLI Upload
Upload a single file or a whole directory:
```bash
# Upload a directory into an album named "Summer 2024"
./gpmc /path/to/photos --album "Summer 2024" --recursive --progress

# Upload with specific filters
./gpmc /path/to/photos --filter ".*\.jpg$" --regex --threads 10
```

### Web UI
Launch the interactive dashboard:
```bash
./gpmc serve --port 8080
```
Open `http://localhost:8080` in your browser to access the premium upload interface.

### Cache Management
Update your local library cache to speed up duplicate detection:
```bash
./gpmc update-cache
```

## 🖥 Web Dashboard Preview

The Web UI features:
- **Glassmorphism Design**: A modern, sleek interface built with Tailwind CSS.
- **Dynamic Progress Tracking**: Real-time Server-Sent Events (SSE) for monitoring active uploads.
- **Batch Processing**: Drag-and-drop support for large folders.
- **Theme Switcher**: Seamless transition between Light and Dark modes.

## 📄 License

[MIT](LICENSE) - See the LICENSE file for details.

---
*Disclaimer: This project is for educational purposes only. Use it at your own risk. Respect Google's Terms of Service.*
