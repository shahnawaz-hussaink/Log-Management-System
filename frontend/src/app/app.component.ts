import { Component, OnInit, OnDestroy, HostListener } from '@angular/core';
import { RouterOutlet, Router } from '@angular/router';
import { AuthService } from './services/auth.service';
import { ApiService } from './services/api.service';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, CommonModule, FormsModule],
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent implements OnInit, OnDestroy {
  notifications: any[] = [];
  showNotificationsDropdown: boolean = false;
  showProfileDropdown: boolean = false;
  showMobileMenu: boolean = false;
  unreadCount: number = 0;
  searchQuery: string = '';
  activeTab: string = 'pending_me';
  private intervalId: any;
 
  // Compose Email Modal States
  showComposeEmailModal: boolean = false;
  emailRecipients: any[] = [];
  composeTo: string = '';
  composeSubject: string = '';
  composeBody: string = '';
  sendingEmail: boolean = false;
  emailSendError: string = '';
  emailSendSuccess: string = '';

  // Sidebar & Category States
  isSidebarCollapsed: boolean = false;
  activeAdminSection: string = 'overview';
  documentTypes: any[] = [];
  selectedFolder: string = 'All';
  viewMode: string = 'receipts';

  constructor(
    public authService: AuthService,
    private api: ApiService,
    public router: Router
  ) {}

  ngOnInit() {
    this.authService.currentUser$.subscribe(user => {
      if (user) {
        this.startNotificationsPolling();
        this.loadDocumentTypes();
      } else {
        this.stopNotificationsPolling();
        this.documentTypes = [];
      }
    });

    this.api.activeTabSubject.subscribe(tab => {
      this.activeTab = tab;
    });

    this.api.searchSubject.subscribe(q => {
      this.searchQuery = q;
    });

    this.router.events.subscribe(() => {
      const url = this.router.url;
      if (url.includes('mode=files')) {
        this.viewMode = 'files';
      } else if (url.includes('mode=receipts')) {
        this.viewMode = 'receipts';
      } else {
        this.viewMode = 'receipts';
      }
    });
  }

  ngOnDestroy() {
    this.stopNotificationsPolling();
  }

  startNotificationsPolling() {
    this.loadNotifications();
    this.intervalId = setInterval(() => {
      this.loadNotifications();
    }, 10000); // Poll every 10 seconds
  }

  stopNotificationsPolling() {
    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
    this.notifications = [];
    this.unreadCount = 0;
  }

  loadNotifications() {
    this.api.getNotifications().subscribe({
      next: (notifs) => {
        this.notifications = notifs || [];
        this.unreadCount = this.notifications.filter(n => n.Status === 'pending').length;
      }
    });
  }

  toggleNotifications() {
    this.showNotificationsDropdown = !this.showNotificationsDropdown;
    if (this.showNotificationsDropdown) {
      // Mark all read visually
      this.unreadCount = 0;
    }
  }

  getNotificationText(n: any): string {
    try {
      const payload = JSON.parse(n.Payload);
      if (n.Template === 'action_required') {
        const sender = payload.uploader_name || payload.actor_name || 'Staff';
        return `Action required: "${payload.document_title}" submitted by ${sender}`;
      } else if (n.Template === 'approved') {
        return `Document approved: "${payload.document_title}" approved by ${payload.actor_name}`;
      } else if (n.Template === 'rejected') {
        return `Document rejected: "${payload.document_title}" rejected by ${payload.actor_name}`;
      } else if (n.Template === 'sent_back') {
        return `Document sent back for revision: "${payload.document_title}" by ${payload.actor_name}`;
      } else if (n.Template === 'closed') {
        return `Your file "${payload.document_title}" has been signed and closed by ${payload.actor_name}`;
      } else if (n.Template === 'sla_warning') {
        return payload.message || `SLA Warning: "${payload.document_title}" has breached deadline.`;
      }
      return `Update on Document ID: ${n.DocumentID}`;
    } catch (e) {
      return `New document update event received.`;
    }
  }

  onSearchInput(event: any) {
    const val = event.target.value;
    this.searchQuery = val;
    this.api.searchSubject.next(val);
    if (this.router.url !== '/dashboard') {
      this.router.navigate(['/dashboard']);
    }
  }

  selectTab(tab: string) {
    this.api.activeTabSubject.next(tab);
    if (this.router.url !== '/dashboard') {
      this.router.navigate(['/dashboard']);
    }
    this.showMobileMenu = false;
  }

  toggleProfileDropdown() {
    this.showProfileDropdown = !this.showProfileDropdown;
  }

  toggleMobileMenu() {
    this.showMobileMenu = !this.showMobileMenu;
  }

  @HostListener('document:click')
  onDocumentClick() {
    this.showProfileDropdown = false;
    this.showNotificationsDropdown = false;
  }

  onNotificationClick(n: any) {
    if (n.DocumentID) {
      this.router.navigate(['/details', n.DocumentID]);
    }
    this.showNotificationsDropdown = false;
  }
 
  openComposeEmailModal() {
    this.showComposeEmailModal = true;
    this.emailSendError = '';
    this.emailSendSuccess = '';
    this.composeTo = '';
    this.composeSubject = '';
    this.composeBody = '';
    this.api.getUsers().subscribe({
      next: (users) => {
        this.emailRecipients = users || [];
      },
      error: (err) => {
        console.error('Failed to load users for email compose modal:', err);
      }
    });
  }
 
  closeComposeEmailModal() {
    this.showComposeEmailModal = false;
  }
 
  openMailto(event: Event) {
    event.preventDefault();
    if (!this.composeTo) return;
    
    let mailtoUrl = `mailto:${this.composeTo}`;
    const params: string[] = [];
    if (this.composeSubject) {
      params.push(`subject=${encodeURIComponent(this.composeSubject)}`);
    }
    if (this.composeBody) {
      params.push(`body=${encodeURIComponent(this.composeBody)}`);
    }
    if (params.length > 0) {
      mailtoUrl += `?${params.join('&')}`;
    }
    
    window.location.href = mailtoUrl;
    this.closeComposeEmailModal();
  }

  logout() {
    this.stopNotificationsPolling();
    this.authService.logout();
    this.router.navigate(['/login']);
  }

  // Sidebar navigation helpers
  loadDocumentTypes() {
    this.api.getDocumentTypes().subscribe({
      next: (types) => {
        this.documentTypes = types || [];
      },
      error: (err) => console.error('Failed to load document types in App:', err)
    });
  }

  toggleTheme() {
    document.documentElement.classList.toggle('dark');
  }

  toggleSidebar() {
    this.isSidebarCollapsed = !this.isSidebarCollapsed;
  }

  isActiveDashboard(): boolean {
    return this.router.url.startsWith('/dashboard');
  }

  navigateToDashboard() {
    this.selectedFolder = 'All';
    this.viewMode = 'receipts';
    this.api.activeTabSubject.next('all_files');
    this.router.navigate(['/dashboard'], { queryParams: { folder: 'All', mode: 'receipts' } });
  }

  navigateToMode(mode: 'receipts' | 'files') {
    this.selectedFolder = 'All';
    this.viewMode = mode;
    this.api.activeTabSubject.next('all_files');
    this.router.navigate(['/dashboard'], { queryParams: { folder: 'All', mode: mode } });
  }

  navigateToAdminSection(section: string) {
    this.activeAdminSection = section;
    this.router.navigate(['/admin'], { queryParams: { section } });
  }

  selectSidebarFolder(folder: string) {
    this.selectedFolder = folder;
    this.router.navigate(['/dashboard'], { queryParams: { folder } });
  }

  getCategoryIcon(name: string): string {
    return '';
  }
}
