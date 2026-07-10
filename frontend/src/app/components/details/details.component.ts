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
  styleUrls: ['./details.component.css']
})
export class DetailsComponent implements OnInit {
  document: any = null;
  history: any[] = [];
  currentUser: any = null;
  
  actionRemarks: string = '';
  selectedUser: string = '';
  users: any[] = [];
  
  selectedFile: File | null = null;
  replaceError: string = '';
  replaceRemarks: string = '';

  showSignModal: boolean = false;
  pdfCacheBuster: number = Date.now();
  pendingAction: string = '';
  signMode: 'draw' | 'type' = 'draw';
  typedName: string = '';

  private isDrawing: boolean = false;
  private ctx: CanvasRenderingContext2D | null = null;

  constructor(
    private route: ActivatedRoute,
    private api: ApiService,
    private auth: AuthService,
    public router: Router,
    private sanitizer: DomSanitizer
  ) {}

  ngOnInit() {
    this.currentUser = this.auth.getCurrentUser();
    if (!this.currentUser) {
      this.router.navigate(['/login']);
      return;
    }

    this.api.getUsers().subscribe(res => {
      this.users = res.filter(u => u.ID !== this.currentUser.ID);
      if (this.users.length > 0) {
        this.selectedUser = this.users[0].ID;
      }
    });

    this.route.paramMap.subscribe(params => {
      const id = params.get('id');
      if (id) {
        this.loadDetails(id);
      }
    });
  }

  loadDetails(id: string) {
    this.api.getDocumentDetails(id).subscribe(res => {
      this.document = res.document;
      this.history = res.history;
      this.pdfCacheBuster = Date.now();
      
      if (this.isDocx(this.document.Filename)) {
        setTimeout(() => {
          this.renderDocxPreview();
        }, 100);
      }
    });
  }

  download() {
    const token = this.auth.getToken();
    window.open(`http://localhost:8080/api/documents/${this.document.ID}/download?token=${token}`, '_blank');
  }

  openSignModal(action: string) {
    this.pendingAction = action;
    this.showSignModal = true;
    this.typedName = this.currentUser.Name;
    setTimeout(() => {
      this.initCanvas();
    }, 50);
  }

  setSignMode(mode: 'draw' | 'type') {
    this.signMode = mode;
    setTimeout(() => {
      this.initCanvas();
    }, 50);
  }

  initCanvas() {
    const canvas = document.getElementById('sig-canvas') as HTMLCanvasElement | null;
    if (!canvas) return;
    canvas.setAttribute('width', '450');
    canvas.setAttribute('height', '180');
    this.ctx = canvas.getContext('2d');
    if (!this.ctx) return;
    
    this.ctx.strokeStyle = '#1e3a8a';
    this.ctx.lineWidth = 3;
    this.ctx.lineCap = 'round';
    
    if (this.signMode === 'type') {
      this.drawTypedSignature();
    }
  }

  drawTypedSignature() {
    if (!this.ctx) return;
    this.ctx.clearRect(0, 0, 450, 180);
    this.ctx.font = 'italic 36px "Brush Script MT", cursive, "Dancing Script"';
    this.ctx.fillStyle = '#1e3a8a';
    this.ctx.textAlign = 'center';
    this.ctx.textBaseline = 'middle';
    this.ctx.fillText(this.typedName || this.currentUser.Name, 225, 90);
  }

  startDrawing(event: MouseEvent | TouchEvent) {
    if (this.signMode !== 'draw') return;
    this.isDrawing = true;
    const pos = this.getEventPos(event);
    if (this.ctx && pos) {
      this.ctx.beginPath();
      this.ctx.moveTo(pos.x, pos.y);
    }
    event.preventDefault();
  }

  draw(event: MouseEvent | TouchEvent) {
    if (!this.isDrawing || this.signMode !== 'draw' || !this.ctx) return;
    const pos = this.getEventPos(event);
    if (pos) {
      this.ctx.lineTo(pos.x, pos.y);
      this.ctx.stroke();
    }
    event.preventDefault();
  }

  stopDrawing() {
    this.isDrawing = false;
  }

  clearCanvas() {
    if (this.ctx) {
      this.ctx.clearRect(0, 0, 450, 180);
      if (this.signMode === 'type') {
        this.drawTypedSignature();
      }
    }
  }

  private getEventPos(event: MouseEvent | TouchEvent): { x: number, y: number } | null {
    const canvas = document.getElementById('sig-canvas');
    if (!canvas) return null;
    const rect = canvas.getBoundingClientRect();
    
    let clientX = 0;
    let clientY = 0;
    
    if (event instanceof MouseEvent) {
      clientX = event.clientX;
      clientY = event.clientY;
    } else if (event.touches && event.touches.length > 0) {
      clientX = event.touches[0].clientX;
      clientY = event.touches[0].clientY;
    } else {
      return null;
    }
    
    return {
      x: clientX - rect.left,
      y: clientY - rect.top
    };
  }

  submitAction(action: string) {
    if (action === 'Sent Back') {
      this.executeSubmitAction(action, '');
    } else {
      this.openSignModal(action);
    }
  }

  confirmSignature() {
    const canvas = document.getElementById('sig-canvas') as HTMLCanvasElement | null;
    if (!canvas) return;
    
    const signatureBase64 = canvas.toDataURL('image/png');
    this.showSignModal = false;
    this.executeSubmitAction(this.pendingAction, signatureBase64);
  }

  executeSubmitAction(action: string, signature: string) {
    let target = null;
    if (action === 'Sent Back' || action === 'Rejected') {
      target = this.document.UploaderID;
    } else if (action === 'Approved') {
      target = this.currentUser.ID; // or specific user
    } else if (action === 'Forwarded') {
      target = this.selectedUser;
    }

    this.api.submitAction(this.document.ID, {
      actor_id: this.currentUser.ID,
      target_id: target,
      action: action,
      remarks: this.actionRemarks,
      signature: signature
    }).subscribe(() => {
      this.loadDetails(this.document.ID);
      this.actionRemarks = '';
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

  renderDocxPreview() {
    if (!this.document) return;
    const token = this.auth.getToken();
    const url = `http://localhost:8080/api/documents/${this.document.ID}/download?token=${token}&cb=${this.pdfCacheBuster}`;
    
    fetch(url)
      .then(response => response.blob())
      .then(blob => {
        const container = document.getElementById('docx-container');
        if (container) {
          container.innerHTML = '';
          import('docx-preview').then(docx => {
            docx.renderAsync(blob, container, undefined, {
              className: 'docx-rendered',
              inWrapper: true,
              ignoreWidth: true,
              ignoreHeight: true,
              ignoreFonts: false,
              breakPages: false,
              debug: false,
              trimXmlDeclaration: true
            }).catch(err => {
              console.error('Docx render error:', err);
              container.innerHTML = `<div class="flex items-center justify-center h-full text-rose-500 font-semibold p-6 text-center border-2 border-dashed border-rose-200 rounded-xl bg-rose-50/50">
                <p>Failed to render preview. The document might be too large or complex for the browser previewer. Please use the download button below to view it natively.</p>
              </div>`;
            });
          });
        }
      })
      .catch(err => {
        console.error('Error fetching docx:', err);
      });
  }

  getPdfUrl(): SafeResourceUrl {
    if (!this.document) return '';
    const token = this.auth.getToken();
    const url = `http://localhost:8080/api/documents/${this.document.ID}/download?token=${token}&cb=${this.pdfCacheBuster}`;
    return this.sanitizer.bypassSecurityTrustResourceUrl(url);
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

    this.api.replaceDocument(this.document.ID, formData).subscribe({
      next: () => {
        this.loadDetails(this.document.ID);
        this.selectedFile = null;
        this.replaceRemarks = '';
        this.replaceError = '';
        const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
        if (fileInput) fileInput.value = '';
      },
      error: () => {
        this.replaceError = 'Failed to resubmit document.';
      }
    });
  }
}
