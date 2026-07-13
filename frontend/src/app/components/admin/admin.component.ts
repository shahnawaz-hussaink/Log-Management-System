import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './admin.component.html',
  styleUrls: ['./admin.component.css']
})
export class AdminComponent implements OnInit {
  currentUser: any = {};
  activeSection: string = 'overview';
  isSuperAdmin: boolean = false;  // true when at /superadmin
  loadingStats: boolean = false;
  loadingUsers: boolean = false;
  loadingDocTypes: boolean = false;
  loadingSchools: boolean = false;

  // Stats
  stats: any = {};

  // Users
  users: any[] = [];
  filteredUsers: any[] = [];
  userSearch: string = '';
  showUserModal: boolean = false;
  editingUser: any = null;
  userForm: any = { name: '', email: '', role: 'Student', password: '', class_section: '', subject: '', phone: '', school_id: null };
  userError: string = '';
  userSuccess: string = '';
  deleteConfirmUserId: string = '';

  // Document Types
  docTypes: any[] = [];
  filteredDocTypes: any[] = [];
  docTypeSearch: string = '';
  showDocTypeModal: boolean = false;
  editingDocType: any = null;
  docTypeForm: any = { name: '', slug: '', workflow_stages: '[]', required_fields: '[]', sla_hours: 72, needs_parent_cosign: false, active: true };
  docTypeError: string = '';
  docTypeSuccess: string = '';
  deleteConfirmDocTypeId: string = '';

  // Schools
  schools: any[] = [];
  showSchoolModal: boolean = false;
  editingSchool: any = null;
  schoolForm: any = { name: '', slug: '', settings: '' };
  schoolError: string = '';
  schoolSuccess: string = '';

  roles = ['Student', 'Teacher', 'Principal', 'Admin', 'Parent'];

  constructor(
    private api: ApiService,
    private auth: AuthService,
    private router: Router,
    private route: ActivatedRoute
  ) {}

  ngOnInit() {
    this.currentUser = this.auth.getCurrentUser() || {};
    const role = this.currentUser.Role || this.currentUser.role;
    this.isSuperAdmin = (role === 'Admin' || role === 'SuperAdmin');
    this.loadStats();
    this.loadUsers();
    this.loadDocTypes();
    if (this.isSuperAdmin) {
      this.loadSchools();
    }

    // Subscribe to query params for global sidebar navigation
    this.route.queryParams.subscribe(params => {
      if (params['section']) {
        this.activeSection = params['section'];
        this.clearMessages();
      }
    });
  }

  setSection(section: string) {
    this.activeSection = section;
    this.clearMessages();
  }

  clearMessages() {
    this.userError = ''; this.userSuccess = '';
    this.docTypeError = ''; this.docTypeSuccess = '';
    this.schoolError = ''; this.schoolSuccess = '';
  }

  logout() {
    this.auth.logout();
    this.router.navigate(['/login']);
  }

  goToDashboard() {
    this.router.navigate(['/dashboard']);
  }

  // ── Stats ──────────────────────────────────────────────────────────────────

  loadStats() {
    this.loadingStats = true;
    this.api.getAdminStats().subscribe({
      next: (data) => {
        this.stats = data;
        this.loadingStats = false;
      },
      error: () => { this.loadingStats = false; }
    });
  }

  // ── Users ──────────────────────────────────────────────────────────────────

  loadUsers() {
    this.loadingUsers = true;
    this.api.getAdminUsers().subscribe({
      next: (data) => {
        this.users = data || [];
        this.applyUserFilter();
        this.loadingUsers = false;
      },
      error: () => { this.loadingUsers = false; }
    });
  }

  applyUserFilter() {
    const q = this.userSearch.toLowerCase();
    this.filteredUsers = q
      ? this.users.filter(u => u.Name?.toLowerCase().includes(q) || u.Email?.toLowerCase().includes(q) || u.Role?.toLowerCase().includes(q))
      : [...this.users];
  }

  openCreateUser() {
    this.editingUser = null;
    this.userForm = { name: '', email: '', role: 'Student', password: '', class_section: '', subject: '', phone: '', school_id: this.schools[0]?.ID || null };
    this.userError = '';
    this.showUserModal = true;
  }

  openEditUser(user: any) {
    this.editingUser = user;
    this.userForm = {
      name: user.Name,
      email: user.Email,
      role: user.Role,
      password: '',
      class_section: user.ClassSection || '',
      subject: user.Subject || '',
      phone: user.Phone || '',
      school_id: user.SchoolID
    };
    this.userError = '';
    this.showUserModal = true;
  }

  saveUser() {
    this.userError = '';
    const nameTrimmed = this.userForm.name.trim();
    const emailTrimmed = this.userForm.email.trim().toLowerCase();

    if (!nameTrimmed || !emailTrimmed) {
      this.userError = 'Name and email are required.';
      return;
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(emailTrimmed)) {
      this.userError = 'Please enter a valid email address.';
      return;
    }

    const payload = { ...this.userForm, name: nameTrimmed, email: emailTrimmed };
    
    // Default password to "password" if blank for a new user creation
    if (!this.editingUser && !payload.password) {
      payload.password = 'password';
    }

    if (this.editingUser) {
      this.api.adminUpdateUser(this.editingUser.ID, payload).subscribe({
        next: () => {
          this.showUserModal = false;
          this.userSuccess = 'User updated successfully.';
          this.loadUsers();
          this.loadStats();
          setTimeout(() => this.userSuccess = '', 3000);
        },
        error: (e) => this.userError = e.error?.error || 'Failed to update user.'
      });
    } else {
      this.api.adminCreateUser(payload).subscribe({
        next: () => {
          this.showUserModal = false;
          this.userSuccess = 'User created successfully.';
          this.loadUsers();
          this.loadStats();
          setTimeout(() => this.userSuccess = '', 3000);
        },
        error: (e) => this.userError = e.error?.error || 'Failed to create user.'
      });
    }
  }

  confirmDeleteUser(id: string) {
    this.deleteConfirmUserId = id;
  }

  deleteUser(id: string) {
    this.api.adminDeleteUser(id).subscribe({
      next: () => {
        this.deleteConfirmUserId = '';
        this.userSuccess = 'User deleted.';
        this.loadUsers();
        this.loadStats();
        setTimeout(() => this.userSuccess = '', 3000);
      },
      error: () => this.userError = 'Failed to delete user.'
    });
  }

  getRoleBadgeClass(role: string): string {
    const map: any = {
      'SuperAdmin': 'badge-superadmin',
      'Admin': 'badge-admin',
      'Principal': 'badge-principal',
      'Teacher': 'badge-teacher',
      'Student': 'badge-student',
      'Parent': 'badge-parent'
    };
    return map[role] || 'badge-default';
  }

  // ── Document Types ─────────────────────────────────────────────────────────

  loadDocTypes() {
    this.loadingDocTypes = true;
    this.api.getAdminDocumentTypes().subscribe({
      next: (data) => {
        this.docTypes = data || [];
        this.applyDocTypeFilter();
        this.loadingDocTypes = false;
      },
      error: () => { this.loadingDocTypes = false; }
    });
  }

  applyDocTypeFilter() {
    const q = this.docTypeSearch.toLowerCase();
    this.filteredDocTypes = q
      ? this.docTypes.filter(d => d.Name?.toLowerCase().includes(q) || d.SchoolName?.toLowerCase().includes(q))
      : [...this.docTypes];
  }

  openCreateDocType() {
    this.editingDocType = null;
    this.docTypeForm = {
      name: '', slug: '', workflow_stages: '[]', required_fields: '[]',
      sla_hours: 72, needs_parent_cosign: false, active: true,
      school_id: this.schools[0]?.ID || null
    };
    this.docTypeError = '';
    this.showDocTypeModal = true;
  }

  openEditDocType(dt: any) {
    this.editingDocType = dt;
    this.docTypeForm = {
      name: dt.Name,
      slug: dt.Slug,
      workflow_stages: dt.WorkflowStages,
      required_fields: dt.RequiredFields,
      sla_hours: dt.SlaHours,
      needs_parent_cosign: dt.NeedsParentCosign,
      active: dt.Active
    };
    this.docTypeError = '';
    this.showDocTypeModal = true;
  }

  saveDocType() {
    this.docTypeError = '';
    if (!this.docTypeForm.name) {
      this.docTypeError = 'Document type name is required.';
      return;
    }

    // Validate JSON input structures
    try {
      if (this.docTypeForm.workflow_stages) {
        const parsed = JSON.parse(this.docTypeForm.workflow_stages);
        if (!Array.isArray(parsed)) {
          this.docTypeError = 'Workflow Stages must be a valid JSON array, e.g. [{"stage":1,"role":"Teacher","label":"Teacher"}].';
          return;
        }
      }
    } catch (e) {
      this.docTypeError = 'Workflow Stages is not valid JSON.';
      return;
    }

    try {
      if (this.docTypeForm.required_fields) {
        const parsed = JSON.parse(this.docTypeForm.required_fields);
        if (!Array.isArray(parsed)) {
          this.docTypeError = 'Required Fields must be a valid JSON array, e.g. ["from_date","reason"].';
          return;
        }
      }
    } catch (e) {
      this.docTypeError = 'Required Fields is not valid JSON.';
      return;
    }

    if (!this.docTypeForm.slug) {
      this.docTypeForm.slug = this.docTypeForm.name.toLowerCase().replace(/\s+/g, '-');
    }
    const payload = { ...this.docTypeForm };
    if (this.editingDocType) {
      this.api.adminUpdateDocumentType(this.editingDocType.ID, payload).subscribe({
        next: () => {
          this.showDocTypeModal = false;
          this.docTypeSuccess = 'Document type updated.';
          this.loadDocTypes();
          this.loadStats();
          setTimeout(() => this.docTypeSuccess = '', 3000);
        },
        error: (e) => this.docTypeError = e.error?.error || 'Failed to update.'
      });
    } else {
      this.api.adminCreateDocumentType(payload).subscribe({
        next: () => {
          this.showDocTypeModal = false;
          this.docTypeSuccess = 'Document type created.';
          this.loadDocTypes();
          this.loadStats();
          setTimeout(() => this.docTypeSuccess = '', 3000);
        },
        error: (e) => this.docTypeError = e.error?.error || 'Failed to create.'
      });
    }
  }

  confirmDeleteDocType(id: string) {
    this.deleteConfirmDocTypeId = id;
  }

  deleteDocType(id: string) {
    this.api.adminDeleteDocumentType(id).subscribe({
      next: () => {
        this.deleteConfirmDocTypeId = '';
        this.docTypeSuccess = 'Document type deleted.';
        this.loadDocTypes();
        this.loadStats();
        setTimeout(() => this.docTypeSuccess = '', 3000);
      },
      error: () => this.docTypeError = 'Failed to delete.'
    });
  }

  // ── Schools ────────────────────────────────────────────────────────────────

  loadSchools() {
    this.loadingSchools = true;
    this.api.getAdminSchools().subscribe({
      next: (data) => {
        this.schools = data || [];
        this.loadingSchools = false;
      },
      error: () => { this.loadingSchools = false; }
    });
  }

  openEditSchool(school: any) {
    this.editingSchool = school;
    this.schoolForm = { name: school.Name, slug: school.Slug, settings: school.Settings || '' };
    this.schoolError = '';
    this.showSchoolModal = true;
  }

  saveSchool() {
    this.schoolError = '';
    this.api.adminUpdateSchool(this.editingSchool.ID, this.schoolForm).subscribe({
      next: () => {
        this.showSchoolModal = false;
        this.schoolSuccess = 'School updated successfully.';
        this.loadSchools();
        setTimeout(() => this.schoolSuccess = '', 3000);
      },
      error: (e) => this.schoolError = e.error?.error || 'Failed to update school.'
    });
  }

  // ── Helpers ────────────────────────────────────────────────────────────────

  countByRole(role: string): number {
    return this.users.filter(u => u.Role === role).length;
  }

  formatDate(d: string): string {
    if (!d) return '—';
    return new Date(d).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
  }
}
