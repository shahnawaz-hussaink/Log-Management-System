import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
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
  selectedFile: File | null = null;
  targetOwnerId: string = '';
  title: string = '';
  description: string = '';
  category: string = 'Document';
  tags: string = '';
  error: string = '';

  constructor(private api: ApiService, private auth: AuthService, private router: Router) {}

  ngOnInit() {
    if (!this.auth.getCurrentUser()) {
      this.router.navigate(['/login']);
    }
    this.api.getUsers().subscribe(res => {
      this.users = res.filter(u => u.ID !== this.auth.getCurrentUser()?.ID);
      if (this.users.length > 0) {
        this.targetOwnerId = this.users[0].ID;
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

    const formData = new FormData();
    formData.append('file', this.selectedFile);
    formData.append('uploader_id', this.auth.getCurrentUser().ID);
    formData.append('target_owner_id', this.targetOwnerId);
    formData.append('title', this.title);
    formData.append('description', this.description);
    formData.append('category', this.category);
    formData.append('tags', this.tags);

    this.api.uploadDocument(formData).subscribe({
      next: () => {
        this.router.navigate(['/dashboard']);
      },
      error: () => {
        this.error = 'Failed to upload document.';
      }
    });
  }
  
  cancel() {
    this.router.navigate(['/dashboard']);
  }
}
