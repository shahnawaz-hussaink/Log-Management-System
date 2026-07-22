# Log-Management-System

A secure, reactive document sharing and approval workflow application built using a Go (Echo) backend, an Angular frontend, and a PostgreSQL database.

## System Architecture Overview

- **Frontend**: Angular 17+ with Tailwind CSS (Light Office Theme & Micro-animations) running on port `4200`.
- **Backend**: Go 1.26+ (Echo Framework + GORM + JWT Auth) running on port `8080`.
- **Database**: PostgreSQL (Auto-migrations & Seeding via GORM) running on port `5432`.

### Database Schema Diagram
![Database Schema Diagram](http://www.plantuml.com/plantuml/png/fLJVQzim47xtNo6oXsrbWQuqC4efpRfiAwEbPEdOcxbQ5rkBB7btKaDfzxzFiKFin4ePgmeblkz-wBxxTEViW9mlTS8BPIeWA0LPRsHcoI29KSVE1KYxL2ONSz2C7IJJm2mU4n7EHyWMUPtYmcfBujNyUFNNO9Oaqjh-eJwrVKnabslpg3x9doH1uvHx40FFI3mGmAB-LTSv44gA4t5xU_b9d9xUV3ix2yRXCFpZhB1MfrtuSu3h60Cb1lEFyVJYwtZwvcWqUblRiGIbTu0GTslKRu_hs0gObvPaMW0NmNSs-Jch0RAwctHq678sZAICcLFMz53sTxy2rB6_Fwo1gh1cwuPDCa9mA5DmgxgNLvFli_7LJG1y9ID0bpjfCr-6ZuQ_pTV_ShZ9aBg72kCapvN6ED4DbbRWYgtMZuuRr5VQWaIAvyqyqQgoNPfzBkp0UUxH7QZGaUj8v8nKrKoedQUlfvuQNzuXxOneMnGESkwdpS1XRkcXFzq3SO_4JRdBwzdaosIwcZdKRGLcd--RGpdSYcGDgKIGMXdSKKQ93YGuQ1khI5aAckYn8nNjaIamgzp4X4SuHoBG-uTXh5CcJxb0TEqb9C4yT591Xgtn1V9UMbhCjLscXM9dIM6Zyvxv1KkZbNmKXImKY44EACVTe7VYsciDTnhRy1ZpX2NGWbkEcZBHLTFJHLHbUBz-jP5THRY7vjTYYXfMD-H2LPmq-_VfQJYqZ-qo2L63pjCTFEEs-uO7IGit_h7PdRHxh9y_xb_pzQ_Ecr4DadCqqfVwBm00)

### System Architecture Diagram
![System Architecture Diagram](http://www.plantuml.com/plantuml/png/XPHFJ-Cy4CRl_HIFSdXL3hHwsJ-75TkbKA2iG8GsB6UJ3CLRDyxQ3bMruhlln35D24LSh7WylndFyxWziauOLwvKpovbNWWZLaZrgj0vvMxkDeh0XmUjqqAaIx6W-inGwaI-KDJhXJYY4oMPnQNOa45_d2zA9GpcyNWlmjyUWAO1ed5HuEHbQoC82mvL4Ol3mzkBqQXBn9Cpo0U-2I4sz2HfxQauUmZRTxXnwHun9_CaKq9VwLIekTE6hxnNj-NpSFnEXMcInBZ87PcK2aOzDdMkQCNQg5-sq-plZrxr-8Re_m5c938aUabvz3QzMvkHWm5EX58OnnTHxxcb-Z5_K9xv2IlTppM6E4qVzIxLgpXckdJaXmYVVFShcMMAriBGs_5_tYL-03H9M-6Qq0T5V2TCHSuPnlkztNn0db_Fhj3dfRRhsgns0NxfwEQrz8szX9y6g9oSex-MFCqnCMrg3_Qy2Qstr44didhmE3zD0lp3LTXt_2d9RE1_8A66NmB9HRegSR7F0yJ2MC9_-2MKp8GZZtPJcMuqOkxB01DIjE1yp8WxM-Uv9ea9hmydVntWIXIull_D5t_zOiBhht-eBu-4Ro7kXUNZM5ktC3I_kSGa1Btm8Mudnwn_E67r0zynxTPMzqd4lQfXl_VRNTaJdPOh_ceJ77iVkDDc-Q3q6bDfUPfDYmkH1drkRDbASAbJeH24Y7QFC2mBfKLcKPlFDUj9wYW7MOPGOJJgTc8Nl1ijqTYKIjCR_sA6I8p8hKYTrFaE5rjTcFD_5aFS6Ua8Pr9HPUWHvLcvKly0)


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

### Database Seeding / Reset
If you need to reset and populate the database with the pre-seeded mock accounts, navigate to the `backend` folder and run the seeding command:
```bash
go run cmd/seed/main.go
```
This will clear the users database and seed the mock credentials.

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

### System Level / DHE (No School)
- **System Administrator**: `admin@school.edu` (Role: DHE Admin)

### Delhi Public School
- **Gaurav Verma**: `gaurav@school.edu` (Role: School Admin)
- **Neha Reddy**: `neha@school.edu` (Role: Teaching Staff - Department B, History)

### Greenwood High School
- **Rahul Gupta**: `rahul@school.edu` (Role: School Admin)
- **Priya Patel**: `priya@school.edu` (Role: Teaching Staff - Department A, Science)
- **Deepak Singh**: `deepak@school.edu` (Role: Non-teaching)

### Modern School
- **Shalini Sen**: `shalini@school.edu` (Role: School Admin)
- **Vikram Iyer**: `vikram@school.edu` (Role: Teaching Staff - Department C, Mathematics)
- **Meera Menon**: `meera@school.edu` (Role: Teaching Staff - Department D, English)
- **Aarav Sharma**: `aarav@school.edu` (Role: Vocational - Department A)
- **Ananya Iyer**: `ananya@school.edu` (Role: Vocational - Department B)
- **Rohan Das**: `rohan@school.edu` (Role: Vocational - Department C)
- **Kavya Menon**: `kavya@school.edu` (Role: Vocational - Department D)

---

## Key Features Implemented

1. **Receipts (Inward Documents) & Files (Noting Sheets)**:
   - Dual-mode workflow workspace supporting standard inward receipts and official file noting sheets.
2. **Noting Sheet Collaboration (Green & Yellow Notes)**:
   - **Yellow Notes (Drafts)**: Collaborative, editable draft notes that can be edited by the author or finalized.
   - **Green Notes (Official)**: Permanent, un-editable official notes stamped with electronic signatures.
3. **Point of Contact (PoC) Sibling & Parent Routing**:
   - Administrative point-of-contacts can forward files up to parent organizations or across to sibling organizations.
   - Role-based boundaries restrict standard staff visibility to their own organization while admins serve as gateway points of contact.
4. **Document Actions**: Approve, Reject, and Send Back for Revision.
5. **Resubmit or Replace**:
   - Uploaders can replace a document that was sent back.
   - Alternatively, they can **resubmit with comments** without modifying the original file.
6. **Workflow Timeline**: Complete action tracking shown chronologically on a vertical history timeline.
7. **Document Previews (PDF & DOCX)**: 
   - Embeds an inline browser-native PDF viewer dynamically using `DomSanitizer` inside the document details page.
   - Embeds a client-side DOCX document viewer using the `docx-preview` library.
8. **Action Stamp Tokens**: Generates and overlays a secure, verifiable transaction token (e.g. `SIG-TX-XXXX`) automatically when an action (Approve/Reject) is completed.
9. **Separate Admin Panel**: A centralized, secure console for school administrators allowing:
   - System stats oversight (users, documents, SLA metrics).
   - CRUD management for users and class settings.
   - CRUD management for document categories and workflow rules.
   - School settings adjustments.
10. **Forwarded Inbox Sections**: Segregates active received inbox documents from previously acted-upon/forwarded items for better workflow visibility.

