import { Component, OnInit, OnDestroy } from '@angular/core';
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

  // Sidebar & Category States
  isSidebarCollapsed: boolean = false;
  activeAdminSection: string = 'overview';
  documentTypes: any[] = [];
  selectedFolder: string = 'All';

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
    this.api.activeTabSubject.next('all_files');
    this.router.navigate(['/dashboard'], { queryParams: { folder: 'All' } });
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
    const n = name.toLowerCase();
    if (n.includes('assign')) return '📝';
    if (n.includes('leave')) return '📅';
    if (n.includes('permis')) return '🔑';
    if (n.includes('transf') || n.includes('certif')) return '🎓';
    if (n.includes('fee') || n.includes('concess')) return '💰';
    return '📁';
  }
}
