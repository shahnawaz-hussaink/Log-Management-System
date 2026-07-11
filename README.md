# Log-Management-System

A secure, reactive document sharing and approval workflow application built using a Go (Echo) backend, an Angular frontend, and a PostgreSQL database.

## System Architecture Overview

- **Frontend**: Angular 17+ with Tailwind CSS (Light Office Theme & Micro-animations) running on port `4200`.
- **Backend**: Go 1.26+ (Echo Framework + GORM + JWT Auth) running on port `8080`.
- **Database**: PostgreSQL (Auto-migrations & Seeding via GORM) running on port `5432`.

### Database Schema Diagram
![Database Schema Diagram](http://www.plantuml.com/plantuml/png/fPFHQzim4CRV_LU8F6sZXQmiC4Of3LksXT9IicJipLojYoqYIzxfH2Xb__lEthouZWQ1FaRVxtIVlY--3Wp4iRQPnIqhzefCMR7IXh1WurqMTiOrG50hyj7fmfD3Kk-p31qRVbIJ6932H3rbIhrHHgiFQhDPVbANb_StLrToU7xUAGvg5EtxVFNnwtprwc8sUzsTOmt6ZWGPtZZpCw9Sq38DXB3tfFCAGYNiUl5ZBK9128mw1sxFbhUBR-lvxRq8iT4fuTV1jFP5hEN72SPbm2QSQmm5cQRDg7VKqbjZCVNKNiiRu6eWlfrUlFkZnkx5kOlr0z8pOdeKI06CfQV74IOQGnbIkmC3nw4oc_vTsrWF6YaV9l3vfbyKYT1nxlSkZRoW0sbqjxL-drSci2U3pyjOtgqNNnQhheypDO_ibqwsjQqDw-cRJFk7WdtIDmELE1uwKtVHMdYmU1YNVezI4nT0kv3y9uuUstqTujK__NFQpaq6iK_-2utI5_8WT_HnXErO3utlJjDzH2UWVMZ2NgFJiRQ_0G00)

### System Architecture Diagram
![System Architecture Diagram](http://www.plantuml.com/plantuml/png/TLLXKnit4Fr-lsB7cMdnXvZOnfXsKXe7nq079WvsoeSCav6bbQqYdwxIEIxjvBztzqxWayV10pFiUs_thVGGxq8fVBbPsomychmGNgp1kqskfFCvUkY-QG0B8H-N5NhXhFFmQb9zDHWZxzrslZBoCSSGeDiPDC22v6A9i_B98dYKfSYN5hFiPwZRATGcDwLnUG3DlnkjqMDUWXLFw4EZbNJhmkKC_PS1l4zzlNtx-Vhy75wXrgTQC-wyGfzW_SLPJvyIDd5vcI1Tx8ip4P6P-HSZqErHxT2mMwEKnQtmo7BzHYjziZyqvVHwSZ30T7ZAw9uC-ZsLc4W8FpGp5gLOY51RqNg8QjzAmlXbsEycfv3YGzbJD6fd8u5_Q69MEY-MzGJ1dzH3GXZs-TEmaCBT8NyI1pkK37pd-Do89xSfOU7-mP35b47hmXeE287euMYKBpTMU9XDH-qM-JeZNx9ApvTHKS35Sw320yGB5m54BQf8ct85C_JVZ4GuiauA6usq4mNHleQQnhAh-NmArtc9Nc91_2RNxCfCdOudO3Ib6s7gyPl1RPDSAFsLQ-XfvzsIVFQOgyePhtdfu1W-o9NRNv9hS4V7XEmXJHErznN2n3_kvr0jkwEaWiQYwvSYAwlkS2LoglbmIBIA6BClFzxTmX7zQ1zYBfrV9ul6mwwlCzrDnKX2Q-f2kVGu-tG3ppbjGP9PQgqMbOeiV0xa1UPYOV6XheoTtAmf1Mdrtlb7RTqMhamW6qyxy3jzQsSLUcZTS2OW-cn1CwWqLOVdgSURmZgX0eoDHydzwavLbE_GYke5N9aYm4_L6Z3XsKTRGvSY-I_ajNGA5GX-R4CxYoc5rr_Yi_Rm-Zm98lmAtGwr5WhcZk9oMD688mbw7RWGri9eUftja5GIKqFiVWSuwRFeyZqAZxuUrivsqJsiZWS1bn_cKVzxoShVuEyDqcbNJtuxose-zOWFAFNOZo4rE-d0LFY0T7X_2cie79ZjiP8MBw4axQ03OxIu90qT5qqFPVO29oBE9mbEEt0h2lYu-8iMo3Ak4Y--gynJ2s9n7ZzLgnp3t8iy2CbtdBIktq6MQ7R2bqNiUs_8oiHOpDvXhlZ_zFy0)

---

## 1. Database Setup (PostgreSQL)

The application requires a PostgreSQL instance running locally.

### Installation
Make sure PostgreSQL is installed and running on your machine:
- **Windows**: Verify that the PostgreSQL service (e.g., `postgresql-x64-18`) is running.
- **Mac/Linux**: Install via Homebrew/APT and start the service:
  ```bash
  brew services start postgresql
  # or
  sudo systemctl start postgresql
  ```

### Creating the Database
1. Open your terminal or Command Prompt.
2. Run `createdb` using the default `postgres` superuser (enter the database password when prompted):
   ```bash
   createdb -U postgres office_files
   ```
   *(Note: The default password configured in the app is `postgres`. If your credentials differ, set the `DATABASE_URL` environment variable.)*

---

## 2. Backend Setup (Go)

The backend uses a microservice architecture consisting of three components and an API gateway:
- **Auth & User Service**: Handles user sessions, registration, and authentication (runs on port `8081`).
- **Document & Workflow Service**: Handles document metadata, file uploads, signing, approvals, and workflow history (runs on port `8082`).
- **API Gateway**: Exposes a unified endpoint on port `8080` and routes incoming requests dynamically.

### Prerequisites
- Go 1.26 or higher.

### Configuration
By default, the microservices connect using:
`host=localhost user=postgres password=postgres dbname=office_files port=5432 sslmode=disable`

To override the connection string, set the `DATABASE_URL` environment variable:
```bash
# Windows (PowerShell)
$env:DATABASE_URL="host=localhost user=your_user password=your_password dbname=office_files port=5432 sslmode=disable"

# Mac/Linux
export DATABASE_URL="host=localhost user=your_user password=your_password dbname=office_files port=5432 sslmode=disable"
```

### Installation & Run
1. Navigate to the `backend` directory:
   ```bash
   cd backend
   ```
2. Start all services together:
   - **On Mac/Linux**: Run the startup shell script:
     ```bash
     chmod +x start_microservices.sh
     ./start_microservices.sh
     ```
   - **On Windows**: Open three separate terminals in the `backend` folder and run:
     ```bash
     # Terminal 1: Auth Service
     go run services/auth/main.go
     
     # Terminal 2: Document & Workflow Service
     go run services/document/main.go
     
     # Terminal 3: API Gateway
     go run services/gateway/main.go
     ```
   The gateway will be available at `http://localhost:8080`.

### Running Tests
To run backend unit tests:
```bash
go test -v ./internal/handlers/...
```

---

## 3. Frontend Setup (Angular)

The Angular frontend provides dashboard controls for document actions, tracking logs, and a PDF viewer.

### Prerequisites
- Node.js v20+ and npm.
- If Node.js is not on your system path, you can run commands pointing directly to your Node path.

### Installation
1. Navigate to the `frontend` directory:
   ```bash
   cd frontend
   ```
2. Install the package dependencies:
   ```bash
   npm install
   ```

### Run
1. Start the Angular local development server:
   ```bash
   npm start
   ```
2. Open your browser and navigate to [http://localhost:4200](http://localhost:4200).

---

## 4. Test Accounts

The database is pre-seeded with mock users. You can log in on the login screen by entering one of these email addresses along with the default password **`password`**:

### Students
- **Alice Smith**: `alice@school.edu` (Class 10-A)

### Teachers
- **Bob Johnson**: `bob@school.edu` (Subject: Science, Class 10-A)
- **Diana Prince**: `diana@school.edu` (Subject: History, Class 10-B)
- **Evan Wright**: `evan@school.edu` (Subject: Mathematics, Class 10-C)
- **Fiona Gallagher**: `fiona@school.edu` (Subject: English, Class 10-D)

### Principals
- **Charlie Brown**: `charlie@school.edu`
- **George Vance**: `george@school.edu`

### Parents
- **David Smith**: `david@school.edu` (Parent of Alice Smith)

---

## Key Features Implemented

1. **Document Actions**: Approve, Reject, and Send Back for Revision.
2. **Resubmit or Replace**:
   - Uploaders can replace a document that was sent back.
   - Alternatively, they can **resubmit with comments** without modifying the original file.
3. **Workflow Timeline**: Complete action tracking shown chronologically on a vertical history timeline.
4. **Document Previews (PDF & DOCX)**: 
   - Embeds an inline browser-native PDF viewer dynamically using `DomSanitizer` inside the document details page.
   - Embeds a client-side DOCX document viewer using the `docx-preview` library.
5. **Electronic Signature Stamping**:
   - Overlays user signature (drawn via HTML5 Canvas) onto PDFs.
   - Uses custom backend parsing to unzip, inject media assets, update relationship mapping and document XML markup, and re-bundle to stamp signature onto DOCX documents.
