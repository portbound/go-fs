document.addEventListener("alpine:init", () => {
	Alpine.data("galleryApp", () => ({
		// --- STATE ---
		filesByDate: {},
		selectedFile: null,
		isLoading: true,
		// Modals
		showPopover: false,
		showFullscreenImage: false,
		fullscreenImageUrl: "",
		showUploadModal: false,
		showNotifications: false,
		// Uploads
		isUploading: false,
		filesToUpload: [],
		// Toasts & Notifications
		toasts: [],
		notifications: [],
		toastIdCounter: 0,
		notificationIdCounter: 0,

		// --- METHODS ---

		// Initialization
		init() {
			if (!localStorage.getItem("jwt")) {
				window.location.href = "/login.html";
				return;
			}
			this.fetchFiles();
		},

		// API Fetch Wrapper
		async authedFetch(url, options = {}) {
			const token = localStorage.getItem("jwt");
			if (!token) {
				this.addToast("Authentication token not found. Redirecting to login.");
				setTimeout(() => (window.location.href = "/login.html"), 2000);
				return Promise.reject(new Error("Unauthorized"));
			}

			const headers = {
				...options.headers,
				Authorization: `Bearer ${token}`,
			};

			try {
				const response = await fetch(`/api${url}`, { ...options, headers });
				if (response.status === 401) {
					localStorage.removeItem("jwt");
					this.addToast("Session expired. Redirecting to login.");
					setTimeout(() => (window.location.href = "/login.html"), 2000);
					return Promise.reject(new Error("Unauthorized"));
				}
				return response;
			} catch (error) {
				this.addToast("Network error. Please check your connection.");
				return Promise.reject(error);
			}
		},

		// Data Fetching & Processing
		async fetchFiles() {
			this.isLoading = true;
			try {
				const response = await this.authedFetch("/files");
				if (!response.ok) throw new Error("Failed to fetch files.");

				const data = (await response.json()) || [];
				if (Array.isArray(data)) {
					this.filesByDate = this.groupFilesByDate(data);
					Object.values(this.filesByDate)
						.flat()
						.forEach((file) => this.loadThumbnail(file));
				}
			} catch (error) {
				if (error.message !== "Unauthorized") {
					this.addToast("Error fetching files: " + error.message);
				}
			} finally {
				this.isLoading = false;
			}
		},

		groupFilesByDate(files) {
			return files.reduce((acc, file) => {
				const uploadDate = new Date(file.uploadDate); // This Date object represents the UTC instant
				// Format this Date object to a local YYYY-MM-DD string for grouping
				const date = uploadDate.toLocaleDateString('en-CA', { // 'en-CA' locale ensures YYYY-MM-DD format
					year: 'numeric',
					month: '2-digit',
					day: '2-digit'
				});

				if (!acc[date]) acc[date] = [];
				file.thumbnailUrl = ""; // Placeholder
				file.fullUrl = ""; // Placeholder
				acc[date].push(file);
				return acc;
			}, {});
		},

		async loadThumbnail(file) {
			if (file.thumbnailUrl) return;
			try {
				const response = await this.authedFetch(`/files/${file.thumbId}`);
				if (!response.ok) throw new Error("Thumbnail fetch failed");
				const blob = await response.blob();
				file.thumbnailUrl = URL.createObjectURL(blob);
			} catch (error) {
				console.error(`Failed to load thumbnail for ${file.name}:`, error);
				this.addToast(`Could not load thumbnail for ${file.name}`);
			}
		},

		// UI Handlers
		async handleFileClick(file) {
			this.selectedFile = file;
			this.showPopover = true;

			if (!file.fullUrl) {
				try {
					const response = await this.authedFetch(`/files/${file.id}`);
					if (!response.ok) throw new Error("Full media fetch failed");
					const blob = await response.blob();
					this.selectedFile.fullUrl = URL.createObjectURL(blob);
				} catch (error) {
					this.addToast(`Could not load preview for ${file.name}`);
				}
			}
		},

		openFullscreen() {
			if (this.selectedFile?.type.startsWith("image/")) {
				this.fullscreenImageUrl = this.selectedFile.fullUrl;
				this.showFullscreenImage = true;
			}
		},

		logout() {
			localStorage.removeItem("jwt");
			this.addToast("You have been logged out.", "success");
			setTimeout(() => (window.location.href = "/login.html"), 1000);
		},

		// File Actions
		async downloadFile(file) {
			try {
				const response = await this.authedFetch(`/files/${file.id}`);
				if (!response.ok) throw new Error(`Server error: ${response.statusText}`);

				const blob = await response.blob();
				const url = window.URL.createObjectURL(blob);
				const a = document.createElement("a");
				a.style.display = "none";
				a.href = url;
				a.download = file.name;
				document.body.appendChild(a);
				a.click();
				window.URL.revokeObjectURL(url);
				document.body.removeChild(a);
			} catch (error) {
				this.addToast(`Download failed: ${error.message}`);
			}
		},

		async deleteFile(file) {
			if (!confirm("Are you sure you want to delete this file? This action cannot be undone.")) return;

			try {
				const response = await this.authedFetch(`/files/${file.id}`, { method: "DELETE" });
				if (!response.ok) throw new Error("Failed to delete file.");

				// Remove from view
				for (const date in this.filesByDate) {
					this.filesByDate[date] = this.filesByDate[date].filter((f) => f.id !== file.id);
					if (this.filesByDate[date].length === 0) {
						delete this.filesByDate[date];
					}
				}
				this.showPopover = false;
				this.selectedFile = null;
				this.addToast(`"${file.name}" deleted successfully.`, "success");
			} catch (error) {
				this.addToast(error.message);
			}
		},

		// Upload Logic
		handleFileSelect(event) {
			this.filesToUpload = Array.from(event.target.files);
		},
		handleDrop(event) {
			this.filesToUpload = Array.from(event.dataTransfer.files);
		},
		async uploadFiles() {
			if (this.filesToUpload.length === 0) return;

			this.isUploading = true;
			const formData = new FormData();
			this.filesToUpload.forEach((file) => formData.append("files", file));

			try {
				const response = await this.authedFetch("/files", {
					method: "POST",
					body: formData,
				});

				if (response.status === 201) {
					const message = this.filesToUpload.length === 1 
						? `"${this.filesToUpload[0].name}" uploaded successfully!` 
						: `${this.filesToUpload.length} files uploaded successfully!`;
					this.addToast(message, "success");
				} else if (response.status === 207) {
					// Multi-Status
					const errorText = await response.text();
					this.addToast(`Some files failed to upload: ${errorText}`);
				} else {
					const errorData = await response.json();
					throw new Error(errorData.error || "Upload failed");
				}

				this.showUploadModal = false;
				this.filesToUpload = [];
				this.fetchFiles();
			} catch (error) {
				this.addToast(error.message);
			} finally {
				this.isUploading = false;
			}
		},

		// Toasts & Notifications
		addToast(message, type = "error", duration = 5000) {
			const id = this.toastIdCounter++;
			this.toasts.push({ id, message, type });
			this.addNotification(message, type);
			setTimeout(() => this.dismissToast(id), duration);
		},
		dismissToast(id) {
			this.toasts = this.toasts.filter((t) => t.id !== id);
		},
		addNotification(message, type) {
			const id = this.notificationIdCounter++;
			this.notifications.unshift({ id, message, type });
		},

		// --- GETTERS & HELPERS ---
		get sortedDates() {
			return Object.keys(this.filesByDate).sort((a, b) => b.localeCompare(a));
		},
		formatDate(dateString) {
			return new Date(dateString).toLocaleDateString(undefined, {
				year: "numeric",
				month: "long",
				day: "numeric",
			});
		},
		formatBytes(bytes, decimals = 2) {
			if (!+bytes) return "0 Bytes";
			const k = 1024;
			const dm = decimals < 0 ? 0 : decimals;
			const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
			const i = Math.floor(Math.log(bytes) / Math.log(k));
			return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
		},
	}));
});
