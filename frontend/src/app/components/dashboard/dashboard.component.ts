import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
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
  loading: boolean = false;
  historyEntries: any[] = [];
  filteredHistoryEntries: any[] = [];
  loadingHistory: boolean = false;

  viewMode: any = 'receipts';
  files: any[] = [];
  showCreateFileModal: boolean = false;
  newFileForm = { title: '', description: '' };
  createFileError: string = '';

  constructor(
    private api: ApiService, 
    private auth: AuthService, 
    public router: Router,
    private route: ActivatedRoute
  ) {}

  ngOnInit() {
    this.currentUser = this.auth.getCurrentUser();
    if (!this.currentUser) {
      this.router.navigate(['/login']);
      return;
    }
    const role = this.currentUser.Role || this.currentUser.role;
    if (role === 'Admin' || role === 'SuperAdmin') {
      this.router.navigate(['/admin']);
      return;
    }
    this.loadDocumentTypes();

    this.api.searchSubject.subscribe(val => {
      this.searchText = val;
      if (this.viewMode === 'receipts') {
        this.loadDocuments();
      } else {
        this.loadFiles();
      }
      if (this.activeTab === 'archived_closed') {
        this.applyHistoryFilter();
      }
    });

    this.api.activeTabSubject.subscribe(tab => {
      this.activeTab = tab;
      if (tab === 'archived_closed') {
        this.loadHistory();
      } else {
        this.applyFilter();
      }
    });

    // Subscribe to folder parameters from global sidebar
    this.route.queryParams.subscribe(params => {
      if (params['mode']) {
        const newMode = params['mode'] as 'receipts' | 'files';
        if (this.viewMode !== newMode) {
          this.viewMode = newMode;
          this.activeTab = 'all_files';
          this.selectedPriority = 'All';
        }
      }
      if (params['folder']) {
        this.selectedFolder = params['folder'];
      }
      this.applyFilter();
      if (this.viewMode === 'receipts') {
        this.loadDocuments();
      } else {
        this.loadFiles();
      }
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

  setViewMode(mode: 'receipts' | 'files') {
    this.viewMode = mode;
    this.activeTab = 'all_files';
    this.selectedPriority = 'All';
    this.applyFilter();
    if (mode === 'receipts') {
      this.loadDocuments();
    } else {
      this.loadFiles();
    }
  }

  loadDocuments() {
    this.loading = true;
    this.api.getDocuments(this.currentUser.ID, this.searchText).subscribe({
      next: (docs) => {
        this.documents = docs || [];
        this.applyFilter();
        this.loading = false;
      },
      error: (err) => {
        console.error('Failed to load documents:', err);
        this.loading = false;
      }
    });
  }

  loadFiles() {
    this.loading = true;
    this.api.listFiles(this.searchText).subscribe({
      next: (files) => {
        this.files = files || [];
        this.applyFilter();
        this.loading = false;
      },
      error: (err) => {
        console.error('Failed to load files:', err);
        this.loading = false;
      }
    });
  }



  onSearchChange() {
    if (this.viewMode === 'receipts') {
      this.loadDocuments();
    } else {
      this.loadFiles();
    }
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
    if (this.viewMode === 'receipts') {
      let list = this.documents;

      // 1. Folder category filter
      if (this.selectedFolder !== 'All') {
        list = list.filter(doc => doc.Category?.toLowerCase() === this.selectedFolder.toLowerCase());
      }

      // 2. Tab filter
      if (this.activeTab !== 'all_files') {
        const currentUserIdLower = (this.currentUser.ID || this.currentUser.id || '').toLowerCase();

        if (this.activeTab === 'received') {
          list = list.filter(doc => (doc.CurrentOwnerID || '').toLowerCase() === currentUserIdLower && doc.Status !== 'Approved' && doc.Status !== 'Rejected' && doc.Status !== 'Closed' && doc.Status !== 'Archived');
        } else if (this.activeTab === 'my_receipts') {
          list = list.filter(doc => (doc.UploaderID || '').toLowerCase() === currentUserIdLower);
        }
      }

      this.filteredDocuments = list;
    } else {
      let list = this.files;

      const currentUserIdLower = (this.currentUser.ID || this.currentUser.id || '').toLowerCase();

      // 1. Priority filter
      if (this.selectedPriority !== 'All') {
        list = list.filter(file => file.Priority?.toLowerCase() === this.selectedPriority.toLowerCase() || (this.selectedPriority === 'Normal' && !file.Priority));
      }

      // 2. Tab filter
      if (this.activeTab !== 'all_files') {
        if (this.activeTab === 'received') {
          list = list.filter(file => (file.CurrentOwnerID || '').toLowerCase() === currentUserIdLower && (file.CreatorID || '').toLowerCase() !== currentUserIdLower && file.Status !== 'Closed' && file.Status !== 'Archived');
        } else if (this.activeTab === 'my_files') {
          list = list.filter(file => (file.CreatorID || '').toLowerCase() === currentUserIdLower);
        }
      }

      this.filteredDocuments = list;
    }
  }

  getFolderCount(folderName: string): number {
    if (this.viewMode !== 'receipts') return 0;
    let list = this.documents;
    if (folderName !== 'All') {
      list = list.filter(doc => doc.Category?.toLowerCase() === folderName.toLowerCase());
    }
    return list.length;
  }

  getTabCount(tab: string): number {
    let list = this.viewMode === 'receipts' ? this.documents : this.files;
    if (!list) return 0;

    if (this.viewMode === 'receipts') {
      if (tab === 'all_files') {
        return list.length;
      }
      const currentUserIdLower = (this.currentUser.ID || this.currentUser.id || '').toLowerCase();

      if (tab === 'received') {
        list = list.filter(doc => (doc.CurrentOwnerID || '').toLowerCase() === currentUserIdLower && doc.Status !== 'Approved' && doc.Status !== 'Rejected' && doc.Status !== 'Closed' && doc.Status !== 'Archived');
      } else if (tab === 'my_receipts') {
        list = list.filter(doc => (doc.UploaderID || '').toLowerCase() === currentUserIdLower);
      } else {
        return 0;
      }
      return list.length;
    } else {
      if (tab === 'all_files') {
        return list.length;
      }
      const currentUserIdLower = (this.currentUser.ID || this.currentUser.id || '').toLowerCase();

      if (tab === 'received') {
        list = list.filter(file => (file.CurrentOwnerID || '').toLowerCase() === currentUserIdLower && (file.CreatorID || '').toLowerCase() !== currentUserIdLower && file.Status !== 'Closed' && file.Status !== 'Archived');
      } else if (tab === 'my_files') {
        list = list.filter(file => (file.CreatorID || '').toLowerCase() === currentUserIdLower);
      } else {
        return 0;
      }
      return list.length;
    }
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

  loadHistory() {
    this.loadingHistory = true;
    this.api.getMyHistory().subscribe({
      next: (entries) => {
        this.historyEntries = entries || [];
        this.applyHistoryFilter();
        this.loadingHistory = false;
      },
      error: (err) => {
        console.error('Failed to load user history:', err);
        this.loadingHistory = false;
      }
    });
  }

  applyHistoryFilter() {
    let list = this.historyEntries;
    if (this.searchText) {
      const searchLower = this.searchText.toLowerCase();
      list = list.filter(entry => 
        (entry.DocumentTitle || '').toLowerCase().includes(searchLower) ||
        (entry.DocumentNum || '').toLowerCase().includes(searchLower) ||
        (entry.Remarks || '').toLowerCase().includes(searchLower) ||
        (entry.Action || '').toLowerCase().includes(searchLower) ||
        (entry.Actor?.Name || '').toLowerCase().includes(searchLower) ||
        (entry.Target?.Name || '').toLowerCase().includes(searchLower)
      );
    }
    this.filteredHistoryEntries = list;
  }

  goToUpload() {
    this.router.navigate(['/upload']);
  }

  goToDetails(id: string, type?: string) {
    if (type === 'file') {
      this.router.navigate(['/details', id], { queryParams: { type: 'file' } });
    } else {
      this.router.navigate(['/details', id]);
    }
  }

  getCategoryDocCount(categoryName: string): number {
    return this.documents.filter(doc => doc.Category?.toLowerCase() === categoryName.toLowerCase()).length;
  }

  getCategoryApprovedCount(categoryName: string): number {
    return this.documents.filter(doc => doc.Category?.toLowerCase() === categoryName.toLowerCase() && doc.Status === 'Approved').length;
  }

  getOtherDocsCount(): number {
    return this.documents.filter(doc => {
      const cat = doc.Category?.toLowerCase() || '';
      return cat !== 'assignment' && cat !== 'leave application';
    }).length;
  }

  getOtherApprovedCount(): number {
    return this.documents.filter(doc => {
      const cat = doc.Category?.toLowerCase() || '';
      return cat !== 'assignment' && cat !== 'leave application' && doc.Status === 'Approved';
    }).length;
  }

  getLauncherColorClass(categoryName: string): string {
    const n = categoryName.toLowerCase();
    if (n.includes('assign')) return 'bg-blue-50 text-blue-650 border border-blue-150';
    if (n.includes('leave')) return 'bg-emerald-50 text-emerald-650 border border-emerald-150';
    if (n.includes('permis')) return 'bg-amber-55/60 text-amber-650 border border-amber-150';
    if (n.includes('transf') || n.includes('certif')) return 'bg-purple-50 text-purple-650 border border-purple-150';
    if (n.includes('fee') || n.includes('concess')) return 'bg-rose-50 text-rose-650 border border-rose-150';
    return 'bg-slate-50 text-slate-650 border border-slate-150';
  }

  getCategoryIcon(name: string): string {
    return '';
  }

  launchNewDoc(categoryName: string) {
    const type = this.documentTypes.find(dt => dt.Name === categoryName);
    const slug = type ? type.Slug : '';
    this.router.navigate(['/upload'], { queryParams: { category: slug } });
  }
}
