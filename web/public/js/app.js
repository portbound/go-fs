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

        init() {
            this.fetchFiles();
        },

        get sortedDates() {
            return Object.keys(this.files).sort((a, b) => b.localeCompare(a));
        },

        formatDate(dateString) {
            const date = new Date(dateString);
            const options = { year: 'numeric', month: 'long', day: 'numeric' };
            return date.toLocaleDateString(undefined, options);
        },

        fetchFiles() {
            this.isLoading = true;
            fetch('/api/files')
                .then(response => {
                    if (!response.ok) {
                        throw new Error(`HTTP error! status: ${response.status}`);
                    }
                    return response.json();
                })
                .then(data => {
                    if (typeof data === 'object' && data !== null) {
                        this.files = data;
                    } else {
                        this.files = {};
                        if (data) {
                             console.error("API did not return an object:", data);
                        }
                    }
                })
                .catch(error => {
                    console.error('Error fetching files:', error);
                    alert('Failed to fetch files. Check the console for more details.');
                })
                .finally(() => {
                    this.isLoading = false;
                });
        },

        handleFileClick(file) {
            this.selectedFile = file;
            this.showPopover = true;
        },

        openFullscreen() {
            if (this.selectedFile && this.selectedFile.type.startsWith('image/')) {
                this.fullscreenImageUrl = `/api/files/${this.selectedFile.id}`;
                this.showFullscreenImage = true;
                this.showPopover = false;
            }
        },

        deleteFile(id) {
            if (!confirm('Are you sure you want to delete this file?')) {
                return;
            }

            fetch(`/api/files/${id}`, { method: 'DELETE' })
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

            fetch('/api/files', {
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
