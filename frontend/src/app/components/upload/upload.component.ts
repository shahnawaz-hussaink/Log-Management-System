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
  selectedApproverId: string = '';
  title: string = '';
  description: string = '';
  category: string = 'Staff Grievance';
  tags: string = '';
  priority: string = 'Normal';
  error: string = '';
  loading: boolean = false;

  targetClasses: string[] = ['All'];
  availableClasses: string[] = ['All', 'Department A', 'Department B', 'Department C', 'Department D'];

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
        if (true) {
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
        const cSec = currentUser.ClassSection || currentUser.class_section || 'Department A';
        // Allow teachers to broadcast to multiple classes by keeping all available, but default to theirs
        this.targetClasses = [cSec];
      }
    }

    this.api.getUsers().subscribe({
      next: (res) => {
        const currentUser = this.auth.getCurrentUser();
        const currentId = currentUser?.ID || currentUser?.id;
        const currentRole = currentUser?.Role || currentUser?.role;
        const currentSchoolId = currentUser?.SchoolID || currentUser?.school_id;
        const canSeeAll = currentRole === 'School Admin' || currentRole === 'SuperAdmin' || currentRole === 'Admin' || currentRole === 'DHE';
        
        this.users = res.filter(u => {
          if ((u.id || u.ID) === currentId) return false;
          if (canSeeAll) return true;
          return (u.SchoolID || u.school_id) === currentSchoolId;
        });
        if (this.users.length > 0) {
          this.selectedApproverId = this.users[0].id || this.users[0].ID;
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
    if (this.selectedFile) {
      if (this.selectedFile.size > 25 * 1024 * 1024) {
        this.error = 'File size exceeds the 25MB limit.';
        this.selectedFile = null;
        event.target.value = '';
        return;
      }
      this.error = '';
      if (!this.title) {
        // Auto-populate title with filename minus extension
        const name = this.selectedFile.name;
        const idx = name.lastIndexOf('.');
        this.title = idx > 0 ? name.substring(0, idx) : name;
      }
    }
  }



  forwardImmediately: boolean = false;

  upload() {
    if (!this.selectedFile && this.category !== 'Assignment Broadcast') {
      this.error = 'Please select a file.';
      return;
    }

    const currentUser = this.auth.getCurrentUser();

    if (this.category !== 'Official Circular' && this.category !== 'Assignment Broadcast' && this.forwardImmediately && !this.selectedApproverId) {
      this.error = 'Please select a recipient to forward to.';
      return;
    }

    const formData = new FormData();
    if (this.selectedFile) {
      formData.append('file', this.selectedFile as Blob);
    }
    formData.append('uploader_id', currentUser.ID || currentUser.id);
    
    if (this.category !== 'Official Circular' && this.category !== 'Assignment Broadcast') {
      const finalApproverId = this.forwardImmediately ? this.selectedApproverId : (currentUser.ID || currentUser.id);
      formData.append('target_owner_ids', finalApproverId);
    }
    formData.append('target_class', this.targetClasses.join(','));

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
        this.error = err.error?.error || 'Failed to upload document. Please try again.';
      }
    });
  }
  
  cancel() {
    this.router.navigate(['/dashboard']);
  }
}
