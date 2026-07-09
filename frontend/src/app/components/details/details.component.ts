import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

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

  constructor(
    private route: ActivatedRoute,
    private api: ApiService,
    private auth: AuthService,
    public router: Router
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
    });
  }

  download() {
    window.open(`http://localhost:8080/api/documents/${this.document.ID}/download`, '_blank');
  }

  submitAction(action: string) {
    let target = null;
    if (action === 'Sent Back' || action === 'Rejected') {
      target = this.document.UploaderID;
    } else if (action === 'Approved') {
      target = this.currentUser.ID; // or specific user
    }

    this.api.submitAction(this.document.ID, {
      actor_id: this.currentUser.ID,
      target_id: target,
      action: action,
      remarks: this.actionRemarks
    }).subscribe(() => {
      this.loadDetails(this.document.ID);
      this.actionRemarks = '';
    });
  }

  onFileSelected(event: any) {
    this.selectedFile = event.target.files[0];
  }

  replaceFile() {
    if (!this.selectedFile) {
      this.replaceError = 'Please select a file';
      return;
    }

    const formData = new FormData();
    formData.append('file', this.selectedFile);
    formData.append('uploader_id', this.currentUser.ID);
    formData.append('target_owner_id', this.selectedUser);

    this.api.replaceDocument(this.document.ID, formData).subscribe({
      next: () => {
        this.loadDetails(this.document.ID);
        this.selectedFile = null;
        this.replaceError = '';
      },
      error: () => {
        this.replaceError = 'Failed to replace file.';
      }
    });
  }
}
