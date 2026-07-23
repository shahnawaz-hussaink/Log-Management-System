import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-central-repository',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './central-repository.component.html',
  styleUrls: ['./central-repository.component.css']
})
export class CentralRepositoryComponent implements OnInit {
  loading = true;
  files: any[] = [];
  filteredFiles: any[] = [];
  pendingRequests: any[] = [];
  currentUser: any = null;
  searchText = '';
  statusFilter: 'all' | 'Closed' | 'Archived' = 'all';
  activeView: 'repository' | 'requests' = 'repository';

  // Request Access Modal state
  showRequestModal = false;
  selectedFile: any = null;
  requestRemarks = '';
  requestError = '';
  requestSuccess = '';

  // Approve Request Modal state
  showApproveModal = false;
  selectedRequest: any = null;
  approveDuration = 2; // Default 2 hours
  approveError = '';

  // Stat Counters
  totalClosed = 0;
  totalArchived = 0;
  totalPending = 0;

  constructor(
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
    this.loadData();
  }

  loadData() {
    this.loading = true;
    this.api.getClosedOrArchivedFiles(this.searchText).subscribe({
      next: (res) => {
        this.files = res || [];
        this.applyFilter();
        this.updateCounters();
        this.loading = false;
      },
      error: (err) => {
        console.error('Failed to load repository files:', err);
        this.loading = false;
      }
    });

    this.loadPendingRequests();
  }

  loadPendingRequests() {
    this.api.getPendingAccessRequests().subscribe({
      next: (res) => {
        this.pendingRequests = res || [];
        this.totalPending = this.pendingRequests.length;
      },
      error: (err) => {
        console.error('Failed to load pending access requests:', err);
      }
    });
  }

  applyFilter() {
    let temp = this.files;
    if (this.statusFilter !== 'all') {
      temp = temp.filter(f => f.Status === this.statusFilter);
    }
    this.filteredFiles = temp;
  }

  onFilterChange(status: 'all' | 'Closed' | 'Archived') {
    this.statusFilter = status;
    this.applyFilter();
  }

  updateCounters() {
    this.totalClosed = this.files.filter(f => f.Status === 'Closed').length;
    this.totalArchived = this.files.filter(f => f.Status === 'Archived').length;
  }

  isAuthority(): boolean {
    if (!this.currentUser) return false;
    const role = this.currentUser.Role || this.currentUser.role;
    const isStdAuth = role === 'SuperAdmin' || role === 'Admin' || role === 'School Admin' || role === 'Principal' || role === 'DHE' || (role && role.startsWith('Admin '));
    return isStdAuth || this.pendingRequests.length > 0;
  }

  onSearch() {
    this.loadData();
  }

  openRequestModal(file: any) {
    this.selectedFile = file;
    this.requestRemarks = '';
    this.requestError = '';
    this.requestSuccess = '';
    this.showRequestModal = true;
  }

  submitAccessRequest() {
    if (!this.requestRemarks.trim()) {
      this.requestError = 'Please provide a reason for requesting access.';
      return;
    }

    this.api.requestFileAccess(this.selectedFile.ID, this.requestRemarks).subscribe({
      next: () => {
        this.requestSuccess = 'Access request submitted successfully to authorities.';
        this.requestError = '';
        setTimeout(() => {
          this.showRequestModal = false;
          this.loadData();
        }, 1500);
      },
      error: (err) => {
        this.requestError = err.error?.error || 'Failed to submit request. Please try again.';
      }
    });
  }

  openApproveModal(req: any) {
    this.selectedRequest = req;
    this.approveDuration = 2; // Reset default to 2 hours
    this.approveError = '';
    this.showApproveModal = true;
  }

  resolveRequest(status: 'approved' | 'rejected') {
    const hours = status === 'approved' ? Number(this.approveDuration) : 0;
    this.api.resolveAccessRequest(this.selectedRequest.ID, status, hours).subscribe({
      next: () => {
        this.showApproveModal = false;
        this.loadPendingRequests();
        this.loadData();
      },
      error: (err) => {
        this.approveError = err.error?.error || 'Failed to resolve request.';
      }
    });
  }

  revokeAccess(file: any) {
    if (confirm(`Are you sure you want to revoke your access to file "${file.Title}"?`)) {
      this.loading = true;
      this.api.revokeFileAccess(file.ID).subscribe({
        next: () => {
          this.loadData();
        },
        error: (err) => {
          console.error('Failed to revoke file access:', err);
          this.loading = false;
          alert(err.error?.error || 'Failed to revoke access.');
        }
      });
    }
  }

  isStandardUserWithAccess(file: any): boolean {
    if (!this.currentUser) return false;
    return file.AccessStatus === 'pending' || file.AccessStatus === 'approved';
  }

  goToDetails(file: any) {
    if (file.HasAccess) {
      this.router.navigate(['/details', file.ID], { queryParams: { type: 'file', source: 'repo' } });
    }
  }
}
