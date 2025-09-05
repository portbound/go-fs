document.addEventListener('alpine:init', () => {
  Alpine.data('fileManager', () => ({
    files: {},
    selectedFile: {},
    showPopover: false,
    showFullscreenImage: false,
    fullscreenImageUrl: '',
    isLoading: true,
    showUploadModal: false,
    isUploading: false,
    filesToUpload: [],

    // A reusable fetch wrapper to add the Authorization header.
    async authedFetch(url, options = {}) {
      const token = localStorage.getItem("jwt");
      if (!token) {
        // Redirect to login if no token is found
        window.location.href = '/login.html';
        return;
      }

      const headers = {
        ...options.headers,
        'Authorization': `Bearer ${token}`,
      };

      const response = await fetch(url, { ...options, headers });

      if (response.status === 401) {
        // Token is invalid or expired, redirect to login
        localStorage.removeItem("jwt");
        window.location.href = '/login.html';
        throw new Error('Unauthorized');
      }

      return response;
    },

    init() {
      this.fetchFiles();
    },

    groupFilesByDate(files) {
      const grouped = {};
      files.forEach(file => {
        const date = new Date(file['uploadDate']).toISOString().split('T')[0]; // Format as YYYY-MM-DD
        if (!grouped[date]) {
          grouped[date] = [];
        }
        // Add placeholder properties for our blob URLs
        file.thumbnailUrl = '';
        file.fullUrl = '';
        grouped[date].push(file);
      });
      return grouped;
    },

    get sortedDates() {
      return Object.keys(this.files).sort((a, b) => b.localeCompare(a));
    },

    formatDate(dateString) {
      const date = new Date(dateString);
      const options = { year: 'numeric', month: 'long', day: 'numeric' };
      return date.toLocaleDateString(undefined, options);
    },

    async loadThumbnail(file) {
      if (file.thumbnailUrl) return; // Already loaded
      try {
        const response = await this.authedFetch(`/api/files/${file.thumbId}`);
        if (!response.ok) throw new Error('Failed to fetch thumbnail');
        const blob = await response.blob();
        file.thumbnailUrl = URL.createObjectURL(blob);
      } catch (error) {
        console.error(`Error loading thumbnail for ${file.name}:`, error);
        // You could set a default broken image URL here
      }
    },

    fetchFiles() {
      this.isLoading = true;
      this.authedFetch('/api/files')
        .then(response => {
          if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
          }
          return response.json();
        })
        .then(data => {
          if (Array.isArray(data)) {
            this.files = this.groupFilesByDate(data);
            // Asynchronously load thumbnails after files are grouped
            Object.values(this.files).flat().forEach(file => this.loadThumbnail(file));
          } else {
            this.files = {};
            if (data) {
              console.error("API did not return an array:", data);
            }
          }
        })
        .catch(error => {
          console.error('Error fetching files:', error);
          if (error.message !== 'Unauthorized') {
            alert('Failed to fetch files. Check the console for more details.');
          }
        })
        .finally(() => {
          this.isLoading = false;
        });
    },

    async handleFileClick(file) {
      this.selectedFile = file;
      this.showPopover = true;

      // Load the full-resolution image/video if it hasn't been loaded yet
      if (!this.selectedFile.fullUrl && (this.selectedFile.type?.startsWith('image/') || this.selectedFile.type?.startsWith('video/'))) {
        try {
          const response = await this.authedFetch(`/api/files/${this.selectedFile.id}`);
          if (!response.ok) throw new Error('Failed to fetch full media');
          const blob = await response.blob();
          this.selectedFile.fullUrl = URL.createObjectURL(blob);
        } catch (error) {
          console.error(`Error loading full media for ${this.selectedFile.name}:`, error);
          alert('Could not load media preview.');
        }
      }
    },

    openFullscreen() {
      if (this.selectedFile && this.selectedFile.type.startsWith('image/')) {
        this.fullscreenImageUrl = this.selectedFile.fullUrl;
        this.showFullscreenImage = true;
        this.showPopover = false;
      }
    },

    async downloadFile(id) {
      try {
        const response = await this.authedFetch(`/api/files/${id}`, { method: 'GET' });
        if (!response.ok) {
          throw new Error(`Failed to download file: ${response.statusText}`);
        }

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.style.display = 'none';
        a.href = url;
        // Use the selectedFile.name for the download attribute
        a.download = this.selectedFile.name || `download_${id}`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
      } catch (error) {
        console.error('Error downloading file:', error);
        alert(error.message);
      }
    },

    deleteFile(id) {
      if (!confirm('Are you sure you want to delete this file?')) {
        return;
      }

      this.authedFetch(`/api/files/${id}`, { method: 'DELETE' })
        .then(response => {
          if (response.ok) {
            // Find and remove the file from the nested structure
            for (const date in this.files) {
              this.files[date] = this.files[date].filter(f => f.id !== id);
              if (this.files[date].length === 0) {
                delete this.files[date];
              }
            }
            this.showPopover = false;
            this.selectedFile = {};
          } else {
            throw new Error('Failed to delete file.');
          }
        })
        .catch(error => {
          console.error('Error deleting file:', error);
          alert(error.message);
        });
    },

    handleFileSelect(event) {
      this.filesToUpload = Array.from(event.target.files);
    },

    handleDrop(event) {
      this.filesToUpload = Array.from(event.dataTransfer.files);
    },



    uploadFiles() {
      if (this.filesToUpload.length === 0) {
        return;
      }

      this.isUploading = true;
      const formData = new FormData();
      this.filesToUpload.forEach(file => {
        formData.append('file', file);
      });

      this.authedFetch('/api/files', {
        method: 'POST',
        body: formData,
      })
        .then(response => {
          if (response.status === 201) {
            return null; // Success with no content to parse
          }
          // For other statuses, try to parse error json
          return response.json().then(data => {
            let errorMessage = 'Upload failed.';
            if (response.status === 207) { // Multi-Status
              errorMessage = data.join('\n');
            } else if (data && data.error) {
              errorMessage = data.error;
            } else if (Array.isArray(data)) {
              errorMessage = data.join('\n');
            }
            throw new Error(errorMessage);
          });
        })
        .then(() => {
          this.showUploadModal = false;
          this.filesToUpload = [];
          this.fetchFiles(); // Refresh file list
        })
        .catch(error => {
          console.error('Error uploading files:', error);
          alert(`Upload failed: ${error.message}`);
        })
        .finally(() => {
          this.isUploading = false;
        });
    },

    formatBytes(bytes, decimals = 2) {
      if (!bytes || bytes === 0) return '0 Bytes';
      const k = 1024;
      const dm = decimals < 0 ? 0 : decimals;
      const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
    }
  }));
});
