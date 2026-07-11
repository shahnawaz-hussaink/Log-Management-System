import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css']
})
export class DashboardComponent implements OnInit {
  documents: any[] = [];
  filteredDocuments: any[] = [];
  currentUser: any = null;
  searchText: string = '';
  documentTypes: any[] = [];
  selectedFolder: string = 'All';
  activeTab: string = 'all_files';
  selectedPriority: string = 'All';
  reportsData: any = null;

  constructor(private api: ApiService, private auth: AuthService, private router: Router) {}

  ngOnInit() {
    this.currentUser = this.auth.getCurrentUser();
    if (!this.currentUser) {
      this.router.navigate(['/login']);
      return;
    }
    this.loadDocumentTypes();

    this.api.searchSubject.subscribe(val => {
      this.searchText = val;
      this.loadDocuments();
    });

    this.api.activeTabSubject.subscribe(tab => {
      this.activeTab = tab;
      this.applyFilter();
    });
    
    if (this.currentUser.Role === 'Admin') {
      this.loadReports();
    }
  }

  loadReports() {
    this.api.getReports().subscribe({
      next: (res) => {
        this.reportsData = res;
      },
      error: (err) => console.error('Failed to load reports:', err)
    });
  }

  loadDocumentTypes() {
    this.api.getDocumentTypes().subscribe({
      next: (types) => {
        this.documentTypes = types;
      }
    });
  }

  loadDocuments() {
    this.api.getDocuments(this.currentUser.ID, this.searchText).subscribe({
      next: (docs) => {
        this.documents = docs || [];
        this.applyFilter();
      }
    });
  }

  onSearchChange() {
    this.loadDocuments();
  }

  selectFolder(folderName: string) {
    this.selectedFolder = folderName;
    this.applyFilter();
  }

  selectTab(tab: string) {
    this.api.activeTabSubject.next(tab);
  }

  selectPriority(priority: string) {
    this.selectedPriority = priority;
    this.applyFilter();
  }

  applyFilter() {
    let list = this.documents;

    // 1. Folder category filter
    if (this.selectedFolder !== 'All') {
      list = list.filter(doc => doc.Category?.toLowerCase() === this.selectedFolder.toLowerCase());
    }

    // 2. Priority filter
    if (this.selectedPriority !== 'All') {
      list = list.filter(doc => doc.Priority?.toLowerCase() === this.selectedPriority.toLowerCase());
    }

    // 3. Tab filter
    if (this.activeTab !== 'all_files') {
      const now = new Date();
      const currentUserIdLower = (this.currentUser.ID || this.currentUser.id || '').toLowerCase();
      
      if (this.currentUser.Role === 'Admin') {
        if (this.activeTab === 'pending_me') {
          list = list.filter(doc => doc.Status !== 'Approved' && doc.Status !== 'Rejected' && doc.Status !== 'Closed' && doc.Status !== 'Archived');
        } else if (this.activeTab === 'sent_out') {
          list = [];
        } else if (this.activeTab === 'overdue') {
          list = list.filter(doc => (doc.Status === 'Pending Approval' || doc.Status === 'Sent Back') && doc.SlaDeadline && new Date(doc.SlaDeadline) < now);
        } else if (this.activeTab === 'approved') {
          list = list.filter(doc => doc.Status === 'Approved');
        } else if (this.activeTab === 'archived_closed') {
          list = list.filter(doc => doc.Status === 'Closed' || doc.Status === 'Archived');
        }
      } else {
        if (this.activeTab === 'pending_me') {
          list = list.filter(doc => (doc.CurrentOwnerID || '').toLowerCase() === currentUserIdLower && doc.Status !== 'Approved' && doc.Status !== 'Rejected' && doc.Status !== 'Closed' && doc.Status !== 'Archived');
        } else if (this.activeTab === 'sent_out') {
          list = list.filter(doc => (doc.UploaderID || '').toLowerCase() === currentUserIdLower && (doc.CurrentOwnerID || '').toLowerCase() !== currentUserIdLower && doc.Status !== 'Approved' && doc.Status !== 'Rejected' && doc.Status !== 'Closed' && doc.Status !== 'Archived');
        } else if (this.activeTab === 'overdue') {
          list = list.filter(doc => (doc.Status === 'Pending Approval' || doc.Status === 'Sent Back') && doc.SlaDeadline && new Date(doc.SlaDeadline) < now);
        } else if (this.activeTab === 'approved') {
          list = list.filter(doc => doc.Status === 'Approved' && ((doc.UploaderID || '').toLowerCase() === currentUserIdLower || (doc.CurrentOwnerID || '').toLowerCase() === currentUserIdLower));
        } else if (this.activeTab === 'archived_closed') {
          list = list.filter(doc => doc.Status === 'Closed' || doc.Status === 'Archived');
        }
      }
    }

    this.filteredDocuments = list;
  }

  getFolderCount(folderName: string): number {
    let list = this.documents;
    if (folderName !== 'All') {
      list = list.filter(doc => doc.Category?.toLowerCase() === folderName.toLowerCase());
    }
    return list.length;
  }

  getTabCount(tab: string): number {
    let list = this.documents;
    if (tab === 'all_files') {
      return list.length;
    }
    const now = new Date();
    const currentUserIdLower = (this.currentUser.ID || this.currentUser.id || '').toLowerCase();

    if (this.currentUser.Role === 'Admin') {
      if (tab === 'pending_me') {
        list = list.filter(doc => doc.Status !== 'Approved' && doc.Status !== 'Rejected' && doc.Status !== 'Closed' && doc.Status !== 'Archived');
      } else if (tab === 'sent_out') {
        list = [];
      } else if (tab === 'overdue') {
        list = list.filter(doc => (doc.Status === 'Pending Approval' || doc.Status === 'Sent Back') && doc.SlaDeadline && new Date(doc.SlaDeadline) < now);
      } else if (tab === 'approved') {
        list = list.filter(doc => doc.Status === 'Approved');
      } else if (tab === 'archived_closed') {
        list = list.filter(doc => doc.Status === 'Closed' || doc.Status === 'Archived');
      }
    } else {
      if (tab === 'pending_me') {
        list = list.filter(doc => (doc.CurrentOwnerID || '').toLowerCase() === currentUserIdLower && doc.Status !== 'Approved' && doc.Status !== 'Rejected' && doc.Status !== 'Closed' && doc.Status !== 'Archived');
      } else if (tab === 'sent_out') {
        list = list.filter(doc => (doc.UploaderID || '').toLowerCase() === currentUserIdLower && (doc.CurrentOwnerID || '').toLowerCase() !== currentUserIdLower && doc.Status !== 'Approved' && doc.Status !== 'Rejected' && doc.Status !== 'Closed' && doc.Status !== 'Archived');
      } else if (tab === 'overdue') {
        list = list.filter(doc => (doc.Status === 'Pending Approval' || doc.Status === 'Sent Back') && doc.SlaDeadline && new Date(doc.SlaDeadline) < now);
      } else if (tab === 'approved') {
        list = list.filter(doc => doc.Status === 'Approved' && ((doc.UploaderID || '').toLowerCase() === currentUserIdLower || (doc.CurrentOwnerID || '').toLowerCase() === currentUserIdLower));
      } else if (tab === 'archived_closed') {
        list = list.filter(doc => doc.Status === 'Closed' || doc.Status === 'Archived');
      }
    }
    return list.length;
  }

  getHoldingDuration(assignedAtStr: string): string {
    if (!assignedAtStr) return 'Unknown';
    const assignedAt = new Date(assignedAtStr);
    const diff = new Date().getTime() - assignedAt.getTime();
    const hours = Math.floor(diff / (1000 * 60 * 60));
    if (hours < 1) {
      const minutes = Math.floor(diff / (1000 * 60));
      return `${minutes}m ago`;
    }
    if (hours > 24) {
      const days = Math.floor(hours / 24);
      return `${days}d ago`;
    }
    return `${hours}h ago`;
  }

  goToUpload() {
    this.router.navigate(['/upload']);
  }

  goToDetails(id: string) {
    this.router.navigate(['/details', id]);
  }
}
