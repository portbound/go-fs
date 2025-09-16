# Go-FS: A Cloud-Native Photo Roll

Go-FS is a self-hosted photo, GIF, and video storage platform designed to provide a seamless, native-like viewing experience for media stored in Google Cloud Storage (GCS). It acts as a "photo roll in the cloud," allowing users to perform standard CRUD opertions on their media.

## The "Why"

This project was born out of curiosity, but ultimately pivoted to solve a problem for my wife: running out of phone storage due to an ever-growing collection of photos and videos. While existing cloud storage solutions are great for backups, they often lack a fluid and intuitive viewing experience for daily browsing. Go-FS bridges this gap, offering a *modest* but performant interface to browse media, without feeling like you've left your phone's native gallery.

Also just trying to learn new things.

## Features

*   **Cloud Storage:** Securely stores all media in a Google Cloud Storage bucket.
*   **Beautiful UI:** A clean, modern, and responsive web interface for browsing media.
*   **CRUD Operations:** Full support for creating (uploading), reading (viewing), updating, and deleting files.
*   **Thumbnail Generation:** Automatically generates thumbnails for faster previews.
*   **Easy Uploading:** Drag-and-drop file uploads.
*   **File Details:** View detailed information for each file, including size, type, and upload date.

## Technologies Used

Go-FS is built with a modern stack, combining the power of Go on the backend with a lightweight and responsive frontend. 
![hey_not_bad](https://github.com/portbound/go-fs/blob/main/Screenshot%20From%202025-09-16%2015-49-36.png)

### Backend

*   **[Go](https://golang.org/)**: The primary language for the backend server and business logic.
*   **[Google Cloud Storage (GCS)](https://cloud.google.com/storage)**: For robust and scalable media storage.
*   **[SQLite](https://www.sqlite.org/index.html)**: Used as the local database for managing file metadata.
*   **[SQLC](https://sqlc.dev/)**: Generates type-safe Go code from SQL queries.

### Frontend

*   **[Alpine.js](https://alpinejs.dev/)**: A rugged, minimal framework for composing JavaScript behavior in your markup.
*   **[Tailwind CSS](https://tailwindcss.com/)**: A utility-first CSS framework for rapid UI development.

## Getting Started

To get a local copy up and running, follow these simple steps.

### Prerequisites

*   Go 1.21 or later
*   A Google Cloud Platform account with a GCS bucket 

### Installation

1.  **Clone the repo**
    ```sh
    git clone https://github.com/your_username/go-fs.git
    cd go-fs
    ```
2.  **Install Go dependencies**
    ```sh
    go mod tidy
    ```
3.  **Set up your environment variables**
    Create a `.env` file in the root of the project and add the required fields. Ref `.env.example`

4.  **Run the server**
    ```sh
    go run cmd/server/main.go
    ```
The application will be available at `http://localhost:8080`.


## License

Distributed under the MIT License. See `LICENSE` for more information.
