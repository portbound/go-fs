# Go-FS: A Cloud-Native Photo Roll

Go-F(ree)S(pace) is a self-hosted photo, GIF, and video storage platform designed to provide a seamless, native-like viewing experience for media stored in Google Cloud Storage (GCS). It acts as a "photo roll in the cloud".

## The "Why"

This project was born out of curiosity, but ultimately I ended up pivoting at least 3 times in an attempt to solve a problem for my wife: running out of phone storage due to an ever-growing collection of photos and videos.

Existing cloud storage solutions are great for backups, but they often lack a fluid and intuitive viewing experience for daily browsing. You kind of just upload things and forget about them. Go-FS is my attempt at bridging that gap, offering a *serviceable*, but performant interface to browse media. I am not a frontend developer lol.

## Features

*   **Cloud Storage:** Securely stores all media in a Google Cloud Storage bucket.
*   **Lightweight UI:** The frontend was built with a lightweight JS framework called AlpineJS. It's pretty minimal, but super snappy.
*   **CRUD Ops:** Upload, download, or delete your images and videos.
*   **Thumbnail Generation:** Automatically generates thumbnails for faster gallery rendering.
*   **Easy Uploading:** Drag-and-drop file uploads.
*   **File Details:** View detailed information for each file, including size, type, and upload date.



Go-FS is built with a an unorthodox stack. I feel like people always say that HTMX is the frontend technology for backend engineers, but I just didn't like how tightly coupled things felt with Templ + HTMX. It was really cool to learn, and it reminded me of ASP.NET, but I just wanted to keep things more loosly coupled so I could plug a new frontend into it at some point if I decide to learn a JS framework for real.
<img width="3005" height="896" alt="image" src="https://github.com/user-attachments/assets/aa28d42b-7f29-4ab6-9075-88afd0ed0f14" />
<img width="3005" height="896" alt="image" src="https://github.com/user-attachments/assets/04399948-09f5-4e6d-9241-d746048776f8" />

## Technologies Used
### Backend

*   **[Go](https://golang.org/)** - The primary language for the backend server and business logic.
*   **[Google Cloud Storage (GCS)](https://cloud.google.com/storage)** - For robust and scalable media storage.
*   **[SQLite](https://www.sqlite.org/index.html)** - Used as the local database for managing file metadata.
*   **[SQLC](https://sqlc.dev/)** - Generates type-safe Go code from SQL queries.
*   **[Docker](https://www.docker.com/)** - Containerization

### Frontend

*   **[Alpine.js](https://alpinejs.dev/)** - A rugged, minimal framework for composing JavaScript behavior in your markup.
*   **[Tailwind CSS](https://tailwindcss.com/)** - A utility-first CSS framework for rapid UI development.

## License

Distributed under the MIT License. See `LICENSE` for more information.
