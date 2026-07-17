import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { DomSanitizer, SafeResourceUrl, SafeHtml } from '@angular/platform-browser';

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

  isFileType: boolean = false;
  file: any = null;
  notes: any[] = [];
  selectedReceiptID: string = '';
  selectedReceipt: any = null;
  activeYellowNote: any = null;
  showAttachReceiptModal: boolean = false;
  availableReceipts: any[] = [];
  attachError: string = '';
  selectedReceiptToAttach: string = '';

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
    public auth: AuthService,
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
        const currentRole = this.currentUser?.Role || this.currentUser?.role;
        const currentSchoolId = this.currentUser?.SchoolID || this.currentUser?.school_id;
        const canSeeAll = currentRole === 'School Admin' || currentRole === 'SuperAdmin' || currentRole === 'Admin' || currentRole === 'DHE';
        
        this.users = res.filter(u => {
          if ((u.id || u.ID) === currentId) return false;
          if (canSeeAll) return true;
          return (u.SchoolID || u.school_id) === currentSchoolId;
        });
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

    this.route.queryParams.subscribe((queryParams) => {
      this.isFileType = queryParams['type'] === 'file';
    });

    this.route.paramMap.subscribe((params) => {
      const id = params.get('id');
      if (id) {
        if (this.isFileType) {
          this.loadFileDetails(id);
        } else {
          this.loadDetails(id);
        }
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

  closeFile() {
    if (!this.file || !this.file.ID) return;
    if (confirm('Are you sure you want to close this file? It will be locked from further edits.')) {
      this.api.closeFile(this.file.ID).subscribe({
        next: (res) => {
          this.file = res;
        },
        error: (err) => {
          console.error('Failed to close file:', err);
          alert(err.error?.error || 'Failed to close file.');
        }
      });
    }
  }

  archiveFile() {
    if (!this.file || !this.file.ID) return;
    if (confirm('Are you sure you want to archive this file? This will permanently retire the file.')) {
      this.api.archiveFile(this.file.ID).subscribe({
        next: (res) => {
          this.file = res;
        },
        error: (err) => {
          console.error('Failed to archive file:', err);
          alert(err.error?.error || 'Failed to archive file.');
        }
      });
    }
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

  loadFileDetails(id: string) {
    this.loading = true;
    this.api.getFileDetails(id).subscribe({
      next: (res) => {
        this.file = res.file;
        this.notes = res.notes || [];
        this.activeYellowNote = this.notes.find(n => n.Type === 'Yellow' && !n.IsDiscarded);
        this.newNote = '';

        // Select the first receipt to preview by default if available
        if (this.file.Receipts && this.file.Receipts.length > 0) {
          this.selectReceipt(this.file.Receipts[0]);
        } else {
          this.selectedReceipt = null;
          this.selectedReceiptID = '';
          this.safePdfUrl = null;
        }
        this.loading = false;
        
        // Also fetch all available (unattached) receipts to support attaching receipts
        this.loadAvailableReceipts();
      },
      error: (err) => {
        console.error('Failed to load file details:', err);
        this.loading = false;
      }
    });
  }

  selectReceipt(receipt: any) {
    this.selectedReceipt = receipt;
    this.selectedReceiptID = receipt.ID;
    this.pdfCacheBuster = Date.now();
    if (receipt.FilePath) {
      const token = this.auth.getToken();
      let url = '';
      if (this.isPdf(receipt.Filename)) {
        url = `http://localhost:8080/api/documents/${receipt.ID}/download?token=${token}&cb=${this.pdfCacheBuster}`;
      } else if (
        this.isDocx(receipt.Filename) ||
        this.isDoc(receipt.Filename)
      ) {
        url = `http://localhost:8080/api/documents/${receipt.ID}/preview-pdf?token=${token}&cb=${this.pdfCacheBuster}`;
      }
      this.safePdfUrl = url
        ? this.sanitizer.bypassSecurityTrustResourceUrl(url)
        : null;
    } else {
      this.safePdfUrl = null;
    }
  }

  loadAvailableReceipts() {
    this.api.getDocuments(this.currentUser.ID).subscribe({
      next: (docs) => {
        // Only show receipts that are not already attached to a file
        this.availableReceipts = (docs || []).filter(doc => !doc.FileID);
        if (this.availableReceipts.length > 0) {
          this.selectedReceiptToAttach = this.availableReceipts[0].ID;
        } else {
          this.selectedReceiptToAttach = '';
        }
      }
    });
  }

  openAttachReceiptModal() {
    this.attachError = '';
    this.loadAvailableReceipts();
    this.showAttachReceiptModal = true;
  }

  attachReceipt() {
    this.attachError = '';
    if (!this.selectedReceiptToAttach) {
      this.attachError = 'Please select a receipt to attach.';
      return;
    }
    this.api.attachReceipt(this.file.ID, this.selectedReceiptToAttach).subscribe({
      next: () => {
        this.showAttachReceiptModal = false;
        this.loadFileDetails(this.file.ID);
      },
      error: (err) => {
        this.attachError = err.error?.error || 'Failed to attach receipt.';
      }
    });
  }

  saveYellowNote() {
    this.noteError = '';
    if (!this.newNote.trim()) {
      this.noteError = 'Note content cannot be empty.';
      return;
    }

    // Always create a new yellow note draft
    this.api.createNote(this.file.ID, this.newNote, 'Yellow').subscribe({
      next: () => {
        this.newNote = '';
        this.loadFileDetails(this.file.ID);
        alert('Yellow note draft created successfully.');
      },
      error: (err) => {
        this.noteError = err.error?.error || 'Failed to add yellow note.';
      }
    });
  }

  startInlineEdit(note: any) {
    note.isEditing = true;
    note.editText = note.Content;
  }

  saveInlineNote(note: any) {
    if (!note.editText || !note.editText.trim()) {
      alert('Note content cannot be empty.');
      return;
    }
    this.api.updateNote(note.ID, note.editText.trim()).subscribe({
      next: () => {
        note.isEditing = false;
        this.loadFileDetails(this.file.ID);
        alert('Yellow note draft updated successfully.');
      },
      error: (err) => {
        alert(err.error?.error || 'Failed to update note.');
      }
    });
  }

  publishYellowNote(note: any) {
    if (!note) return;
    const sig = prompt('Enter your digital token signature prefix (optional, leave blank to auto-generate):');
    if (sig === null) return; // User cancelled

    this.api.publishNote(note.ID, sig).subscribe({
      next: () => {
        this.loadFileDetails(this.file.ID);
        alert('Note published successfully to immutable Green Note.');
      },
      error: (err) => {
        alert(err.error?.error || 'Failed to publish note.');
      }
    });
  }

  submitFileForward() {
    if (!this.selectedUser) {
      alert('Please select a recipient to forward the file.');
      return;
    }
    
    this.api.forwardFile(this.file.ID, this.selectedUser).subscribe({
      next: () => {
        this.showForwardSelect = false;
        this.router.navigate(['/dashboard']);
      },
      error: (err) => {
        alert(err.error?.error || 'Failed to forward file.');
      }
    });
  }

  parseMarkdown(text: string): string {
    if (!text) return '';
    // Escape HTML tags to prevent XSS
    let escaped = text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');

    // Normalize newlines
    escaped = escaped.replace(/\r\n/g, '\n');

    // Parse blockquotes & list items: lines starting with > or * or -
    const lines = escaped.split('\n');
    let insideList = false;
    const parsedLines = lines.map(line => {
      let l = line.trim();
      
      // Horizontal Rule
      if (l === '---' || l === '***' || l === '___') {
        return '<hr class="my-4 border-slate-200 dark:border-slate-800" />';
      }

      // Headers (H1-H4)
      if (l.startsWith('# ')) {
        return `<h1 class="text-xl font-bold text-[var(--text-primary)] mt-3 mb-2">${l.substring(2)}</h1>`;
      }
      if (l.startsWith('## ')) {
        return `<h2 class="text-lg font-bold text-[var(--text-primary)] mt-3 mb-1.5">${l.substring(3)}</h2>`;
      }
      if (l.startsWith('### ')) {
        return `<h3 class="text-base font-bold text-[var(--text-primary)] mt-2 mb-1">${l.substring(4)}</h3>`;
      }
      if (l.startsWith('#### ')) {
        return `<h4 class="text-sm font-semibold text-[var(--text-primary)] mt-2 mb-1">${l.substring(5)}</h4>`;
      }

      // Blockquotes (starts with &gt;)
      if (l.startsWith('&gt; ')) {
        return `<blockquote class="border-l-4 border-indigo-400 pl-3 my-2.5 italic text-[var(--text-secondary)] bg-indigo-50/20 dark:bg-indigo-950/10 py-1 rounded">${l.substring(5)}</blockquote>`;
      }

      // Bullet lists: starts with * or -
      if (l.startsWith('* ') || l.startsWith('- ')) {
        let content = l.substring(2);
        let prefix = '';
        if (!insideList) {
          insideList = true;
          prefix = '<ul class="list-disc pl-5 my-2 space-y-1 text-sm text-[var(--text-secondary)]">';
        }
        return `${prefix}<li>${content}</li>`;
      } else {
        let suffix = '';
        if (insideList) {
          insideList = false;
          suffix = '</ul>';
        }
        return suffix + l;
      }
    });

    if (insideList) {
      parsedLines.push('</ul>');
    }

    let parsedHtml = parsedLines.join('\n');

    // Parse bold: **text**
    parsedHtml = parsedHtml.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');

    // Parse italic: *text*
    parsedHtml = parsedHtml.replace(/\*(.*?)\*/g, '<em>$1</em>');

    // Parse inline code: `code`
    parsedHtml = parsedHtml.replace(/`(.*?)`/g, '<code class="px-1.5 py-0.5 bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-800 rounded text-rose-500 dark:text-rose-400 font-mono text-xs">$1</code>');

    // Convert double newlines to paragraph gaps, and single newlines to br
    parsedHtml = parsedHtml.replace(/\n\n/g, '</p><p class="mt-2">');
    parsedHtml = parsedHtml.replace(/\n/g, '<br />');

    return `<div class="markdown-content"><p>${parsedHtml}</p></div>`;
  }

  renderMarkdown(text: string): SafeHtml {
    const rawHtml = this.parseMarkdown(text);
    return this.sanitizer.bypassSecurityTrustHtml(rawHtml);
  }
}
