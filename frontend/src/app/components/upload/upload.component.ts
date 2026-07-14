import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-upload',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './upload.component.html',
  styleUrls: ['./upload.component.css']
})
export class UploadComponent implements OnInit {
  users: any[] = [];
  documentTypes: any[] = [];
  selectedFile: File | null = null;
  targetOwnerId: string = '';
  title: string = '';
  description: string = '';
  category: string = 'Assignment';
  tags: string = '';
  priority: string = 'Normal';
  error: string = '';
  loading: boolean = false;

  targetClass: string = 'All';
  availableClasses: string[] = ['All', '10-A', '10-B', '10-C', '10-D'];

  constructor(
    private api: ApiService, 
    private auth: AuthService, 
    private router: Router,
    private route: ActivatedRoute
  ) {}

  ngOnInit() {
    if (!this.auth.getCurrentUser()) {
      this.router.navigate(['/login']);
    }

    // Load dynamic document types
    this.api.getDocumentTypes().subscribe({
      next: (res) => {
        const currentUser = this.auth.getCurrentUser();
        if (currentUser && (currentUser.Role === 'Student' || currentUser.Role === 'Parent' || currentUser.role === 'Student' || currentUser.role === 'Parent')) {
          this.documentTypes = res.filter(dt => dt.Name !== 'Circular' && dt.name !== 'Circular');
        } else {
          this.documentTypes = res;
        }

        if (this.documentTypes.length > 0) {
          this.category = this.documentTypes[0].Name;
          
          // Pre-select category if matching slug exists in query parameters
          this.route.queryParams.subscribe(params => {
            if (params['category']) {
              const matched = this.documentTypes.find(dt => dt.Slug === params['category'] || dt.Name.toLowerCase() === params['category'].toLowerCase());
              if (matched) {
                this.category = matched.Name;
              }
            }
          });
        }
      },
      error: (err) => console.error('Failed to load document types:', err)
    });

    const currentUser = this.auth.getCurrentUser();
    if (currentUser) {
      const isTeacher = currentUser.Role === 'Teacher' || currentUser.role === 'Teacher';
      if (isTeacher) {
        const cSec = currentUser.ClassSection || currentUser.class_section || '10-A';
        this.availableClasses = [cSec];
        this.targetClass = cSec;
      }
    }

    this.api.getUsers().subscribe({
      next: (res) => {
        const currentId = this.auth.getCurrentUser()?.ID || this.auth.getCurrentUser()?.id;
        this.users = res.filter(u => (u.id || u.ID) !== currentId && u.Role !== 'Student' && u.role !== 'Student');
        if (this.users.length > 0) {
          this.targetOwnerId = this.users[0].id || this.users[0].ID;
        }
      },
      error: (err) => {
        console.error('Failed to load users:', err);
        this.error = 'Could not load approvers list. Please refresh.';
      }
    });
  }

  onFileSelected(event: any) {
    this.selectedFile = event.target.files[0];
    if (this.selectedFile && !this.title) {
      // Auto-populate title with filename minus extension
      const name = this.selectedFile.name;
      const idx = name.lastIndexOf('.');
      this.title = idx > 0 ? name.substring(0, idx) : name;
    }
  }

  upload() {
    if (!this.selectedFile) {
      this.error = 'Please select a file.';
      return;
    }
    if (!this.targetOwnerId) {
      this.error = 'Please select an approver.';
      return;
    }

    const currentUser = this.auth.getCurrentUser();
    const formData = new FormData();
    formData.append('file', this.selectedFile);
    formData.append('uploader_id', currentUser.ID || currentUser.id);
    
    if (this.category !== 'Circular') {
      formData.append('target_owner_id', this.targetOwnerId);
    }
    formData.append('target_class', this.targetClass);

    formData.append('title', this.title);
    formData.append('description', this.description);
    formData.append('category', this.category);
    formData.append('tags', this.tags);
    formData.append('priority', this.priority);

    this.loading = true;
    this.api.uploadDocument(formData).subscribe({
      next: () => {
        this.loading = false;
        this.router.navigate(['/dashboard']);
      },
      error: (err) => {
        console.error('Failed to upload:', err);
        this.loading = false;
        this.error = 'Failed to upload document. Please try again.';
      }
    });
  }
  
  cancel() {
    this.router.navigate(['/dashboard']);
  }
}
