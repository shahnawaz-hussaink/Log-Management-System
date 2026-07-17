import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';

@Component({
  selector: 'app-details',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './details.component.html',
  styleUrls: ['./details.component.css'],
})
export class DetailsComponent implements OnInit {
  document: any = null;
  history: any[] = [];
  currentUser: any = null;

  actionRemarks: string = '';
  selectedUser: string = '';
  users: any[] = [];
  documentTypes: any[] = [];
  submissions: any[] = [];
  

  selectedFile: File | null = null;
  replaceError: string = '';
  replaceRemarks: string = '';

  newNote: string = '';
  noteError: string = '';
  draftContent: string = '';
  draftError: string = '';
  selectedAttachmentFile: File | null = null;
  attachmentError: string = '';
  referralUser: string = '';

  pdfCacheBuster: number = Date.now();
  safePdfUrl: SafeResourceUrl | null = null;
  showForwardSelect: boolean = false;
  loading: boolean = false;

  constructor(
    private route: ActivatedRoute,
    private api: ApiService,
    private auth: AuthService,
    public router: Router,
    private sanitizer: DomSanitizer,
  ) {}

  toggleForwardSelect() {
    this.showForwardSelect = !this.showForwardSelect;
  }

  ngOnInit() {
    this.currentUser = this.auth.getCurrentUser();
    if (!this.currentUser) {
      this.router.navigate(['/login']);
      return;
    }

    this.api.getUsers().subscribe({
      next: (res) => {
        const currentId = this.currentUser?.ID || this.currentUser?.id;
        this.users = res.filter(
          (u) =>
            (u.id || u.ID) !== currentId &&
            u.Role !== 'Student' &&
            u.role !== 'Student',
        );
        if (this.users.length > 0) {
          this.selectedUser = this.users[0].id || this.users[0].ID;
        }
      },
      error: (err) => console.error('Failed to load users:', err),
    });

    this.api.getDocumentTypes().subscribe({
      next: (types) => {
        this.documentTypes = types || [];
      },
      error: (err) => console.error('Failed to load document types:', err),
    });

    this.route.paramMap.subscribe((params) => {
      const id = params.get('id');
      if (id) {
        this.loadDetails(id);
      }
    });
  }

  loadDetails(id: string) {
    this.loading = true;
    this.api.getDocumentDetails(id).subscribe({
      next: (res) => {
        this.document = res.document;
        this.history = res.history;
        this.pdfCacheBuster = Date.now();
        
        if (this.document.Category === 'Assignment Broadcast') {
          this.loadSubmissions(id);
        }

        if (this.document.FilePath) {
          const token = this.auth.getToken();
          let url = '';
          if (this.isPdf(this.document.Filename)) {
            url = `http://localhost:8080/api/documents/${this.document.ID}/download?token=${token}&cb=${this.pdfCacheBuster}`;
          } else if (
            this.isDocx(this.document.Filename) ||
            this.isDoc(this.document.Filename)
          ) {
            url = `http://localhost:8080/api/documents/${this.document.ID}/preview-pdf?token=${token}&cb=${this.pdfCacheBuster}`;
          }
          this.safePdfUrl = url
            ? this.sanitizer.bypassSecurityTrustResourceUrl(url)
            : null;
        }
        this.loading = false;
      },
      error: (err: any) => {
        console.error('Failed to load document details:', err);
        this.loading = false;
      },
    });
  }

  download() {
    const token = this.auth.getToken();
    window.open(
      `http://localhost:8080/api/documents/${this.document.ID}/download?token=${token}`,
      '_blank',
    );
  }

  submitResponse() {
    if (!this.selectedFile) {
      this.replaceError = 'Please select a file to submit.';
      return;
    }
    const formData = new FormData();
    formData.append('file', this.selectedFile);
    formData.append('uploader_id', this.currentUser.ID || this.currentUser.id);
    // Student submitting responds to the Teacher who created the broadcast
    formData.append('target_owner_ids', this.document.UploaderID);
    formData.append('target_class', this.currentUser.ClassSection || 'All');
    formData.append('title', `Submission for ${this.document.Title}`);
    formData.append('description', this.replaceRemarks || 'Assignment Submission');
    formData.append('category', 'Assignment');
    formData.append('priority', 'Normal');
    formData.append('ref_document_id', this.document.ID);

    this.loading = true;
    this.api.uploadDocument(formData).subscribe({
      next: () => {
        this.loading = false;
        this.replaceError = '';
        this.replaceRemarks = '';
        this.selectedFile = null;
        alert('Assignment response submitted successfully!');
        this.loadSubmissions(this.document.ID); // Reload if needed
      },
      error: (err) => {
        console.error('Failed to submit response:', err);
        this.loading = false;
        this.replaceError = err.error?.error || 'Failed to submit response.';
      }
    });
  }

  takeAction(action: string) {
    this.executeSubmitAction(action, '');
  }

  submitAction(action: string) {
    this.executeSubmitAction(action, '');
  }

  executeSubmitAction(action: string, signature: string) {
    if (
      (action === 'Sent Back' || action === 'Rejected') &&
      !this.actionRemarks.trim()
    ) {
      alert(
        `Please enter your Remarks / Noting Sheet comments for this ${action.toLowerCase()} action.`,
      );
      return;
    }

    let target = null;
    if (action === 'Sent Back' || action === 'Rejected') {
      target = this.document.UploaderID;
    } else if (action === 'Approved') {
      target = this.currentUser.ID; // or specific user
    } else if (action === 'Forwarded') {
      if (!this.selectedUser) {
        alert('Please select a user to forward this document to.');
        return;
      }
      target = this.selectedUser;
    }

    this.api
      .submitAction(this.document.ID, {
        actor_id: this.currentUser.ID,
        target_id: target,
        action: action,
        remarks: this.actionRemarks,
        signature: signature,
      })
      .subscribe({
        next: () => {
          this.loadDetails(this.document.ID);
          this.actionRemarks = '';
          this.showForwardSelect = false;
        },
        error: (err) => {
          console.error('Failed to submit action:', err);
          alert(
            err.error?.message ||
              'Failed to submit action. Please make sure all required fields are filled.',
          );
        },
      });
  }

  loadSubmissions(id: string) {
    this.api.getSubmissions(id).subscribe({
      next: (res) => {
        this.submissions = res || [];
      },
      error: (err) => console.error('Failed to load submissions:', err)
    });
  }

  onFileSelected(event: any) {
    this.selectedFile = event.target.files[0];
  }

  isPdf(filename: string): boolean {
    return filename ? filename.toLowerCase().endsWith('.pdf') : false;
  }

  isDocx(filename: string): boolean {
    return filename ? filename.toLowerCase().endsWith('.docx') : false;
  }

  isDoc(filename: string): boolean {
    return filename ? filename.toLowerCase().endsWith('.doc') : false;
  }

  getPdfUrl(): SafeResourceUrl {
    return this.safePdfUrl || '';
  }

  getSafeSignature(signature: string): any {
    if (!signature) return '';
    return this.sanitizer.bypassSecurityTrustUrl(signature);
  }

  replaceFile() {
    const formData = new FormData();
    if (this.selectedFile) {
      formData.append('file', this.selectedFile);
    }
    formData.append('uploader_id', this.currentUser.ID);
    formData.append('target_owner_id', this.selectedUser);
    formData.append('remarks', this.replaceRemarks);
    formData.append('title', this.document.Title);
    formData.append('description', this.document.Description);
    formData.append('category', this.document.Category);
    formData.append('tags', this.document.Tags);
    formData.append('priority', this.document.Priority);
    formData.append('direction', this.document.Direction);

    this.api.replaceDocument(this.document.ID, formData).subscribe({
      next: (res: any) => {
        this.selectedFile = null;
        this.replaceRemarks = '';
        this.replaceError = '';
        const fileInput = document.querySelector(
          'input[type="file"]',
        ) as HTMLInputElement;
        if (fileInput) fileInput.value = '';

        // Navigate to the newly generated document version
        if (res && (res.ID || res.id)) {
          this.router.navigate(['/details', res.ID || res.id]);
        } else {
          this.loadDetails(this.document.ID);
        }
      },
      error: (err) => {
        this.replaceError = err.error?.error || 'Failed to resubmit document.';
      },
    });
  }

  submitNote() {
    if (!this.newNote.trim()) {
      this.noteError = 'Note content cannot be empty.';
      return;
    }
    this.api.appendNote(this.document.ID, this.newNote).subscribe({
      next: () => {
        this.newNote = '';
        this.noteError = '';
        this.loadDetails(this.document.ID);
      },
      error: (err) => {
        this.noteError = 'Failed to append note to the noting sheet.';
      },
    });
  }

  saveDraft() {
    this.api.saveDraft(this.document.ID, this.draftContent).subscribe({
      next: () => {
        this.draftError = '';
        this.loadDetails(this.document.ID);
        alert('Draft order/letter saved successfully.');
      },
      error: (err) => {
        this.draftError = 'Failed to save draft.';
      },
    });
  }

  onAttachmentSelected(event: any) {
    this.selectedAttachmentFile = event.target.files[0];
  }

  uploadAttachment() {
    if (!this.selectedAttachmentFile) {
      this.attachmentError = 'Please select a file to enclose.';
      return;
    }
    this.api
      .addAttachment(this.document.ID, this.selectedAttachmentFile)
      .subscribe({
        next: () => {
          this.selectedAttachmentFile = null;
          this.attachmentError = '';
          const fileInput = document.getElementById(
            'att-file-input',
          ) as HTMLInputElement;
          if (fileInput) fileInput.value = '';
          this.loadDetails(this.document.ID);
        },
        error: (err) => {
          this.attachmentError =
            err.error?.error || 'Failed to upload attachment.';
        },
      });
  }

  submitReferral(action: string) {
    if (action === 'Refer' && !this.referralUser) {
      alert('Please select a user to refer this document to.');
      return;
    }
    const remarks = prompt(
      `Enter optional remarks for this ${action.toLowerCase()} action:`,
    );
    const actionData = {
      action: action,
      target_id: action === 'Refer' ? this.referralUser : null,
      remarks: remarks || `${action} action completed.`,
    };
    this.api.submitAction(this.document.ID, actionData).subscribe({
      next: () => {
        this.loadDetails(this.document.ID);
      },
      error: (err) => {
        alert(`Failed to complete ${action.toLowerCase()} action.`);
      },
    });
  }

  getDownloadAttachmentUrl(att: any): string {
    const token = this.auth.getToken();
    const id = att.id || att.ID;
    return `http://localhost:8080/api/attachments/${id}/download?token=${token}&cb=${Date.now()}`;
  }

  recallDocument() {
    if (
      confirm(
        'Are you sure you want to recall this document back to your queue?',
      )
    ) {
      this.api.recallDocument(this.document.ID).subscribe({
        next: () => {
          this.loadDetails(this.document.ID);
        },
        error: (err) => {
          alert(
            'Failed to recall document. It may have already been acted on.',
          );
        },
      });
    }
  }
}
