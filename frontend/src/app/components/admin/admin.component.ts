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
  docDashboardTab: string = 'receipt_types';
  isSuperAdmin: boolean = false;  // true when at /superadmin
  loadingStats: boolean = false;
  loadingUsers: boolean = false;
  loadingDocTypes: boolean = false;
  loadingSchools: boolean = false;
  loadingRoles: boolean = false;

  // Stats
  stats: any = {};

  // Roles Management (Hierarchical CRUD)
  dbRoles: any[] = [];
  filteredDbRoles: any[] = [];
  roleSearch: string = '';
  showRoleModal: boolean = false;
  editingRole: any = null;
  roleForm: any = { roleName: '', isAdminAccess: false, parentRole: null };
  roleError: string = '';
  roleSuccess: string = '';
  deleteConfirmRoleId: string = '';

  // Users
  users: any[] = [];
  filteredUsers: any[] = [];
  userSearch: string = '';
  showUserModal: boolean = false;
  editingUser: any = null;
  userForm: any = { name: '', email: '', role: 'vocational', password: '', class_section: '', subject: '', phone: '', school_id: null };
  userError: string = '';
  userSuccess: string = '';
  deleteConfirmUserId: string = '';

  // Document Types (renamed to Receipt Types in user interface)
  docTypes: any[] = [];
  filteredDocTypes: any[] = [];
  docTypeSearch: string = '';
  showDocTypeModal: boolean = false;
  editingDocType: any = null;
  docTypeForm: any = { name: '', slug: '', workflow_stages: '[]', required_fields: '[]', sla_hours: 72,  active: true };
  docTypeError: string = '';
  docTypeSuccess: string = '';
  deleteConfirmDocTypeId: string = '';

  // File Categories
  fileCategories: any[] = [];
  filteredFileCategories: any[] = [];
  fileCategorySearch: string = '';
  showFileCategoryModal: boolean = false;
  editingFileCategory: any = null;
  fileCategoryForm: any = { name: '', slug: '', active: true };
  fileCategorySuccess: string = '';
  fileCategoryError: string = '';
  deleteConfirmFileCategoryId: string = '';

  // File Sub-categories
  fileSubCategories: any[] = [];
  filteredFileSubCategories: any[] = [];
  fileSubCategorySearch: string = '';
  showFileSubCategoryModal: boolean = false;
  editingFileSubCategory: any = null;
  fileSubCategoryForm: any = { category: '', name: '', slug: '', active: true };
  fileSubCategorySuccess: string = '';
  fileSubCategoryError: string = '';
  deleteConfirmFileSubCategoryId: string = '';

  // Schools
  schools: any[] = [];
  showSchoolModal: boolean = false;
  editingSchool: any = null;
  schoolForm: any = { name: '', slug: '', settings: '' };
  schoolError: string = '';
  schoolSuccess: string = '';

  roles = ['SuperAdmin', 'DHE', 'School Admin', 'Teaching staff', 'non-teaching', 'vocational'];

  get parentOrganizations(): any[] {
    return this.schools;
  }

  get availableRoles(): string[] {
    const actorRole = this.currentUser?.Role || this.currentUser?.role;
    if (actorRole === 'SuperAdmin') {
      return Array.from(new Set(this.dbRoles.map(r => r.RoleName)));
    }

    const actorRoleRec = this.dbRoles.find(r => r.RoleName === actorRole);
    if (!actorRoleRec) {
      return [actorRole];
    }

    const allowed = this.dbRoles.filter(r => {
      const isSame = r.RoleName === actorRole;
      const isChild = r.ParentRoleID === actorRoleRec.ID || r.ParentRoleName === actorRole;
      return isSame || isChild;
    });

    return Array.from(new Set(allowed.map(r => r.RoleName)));
  }

  onRoleChange() {
    if (this.userForm.role === 'DHE' || this.userForm.role === 'SuperAdmin') {
      this.userForm.school_id = null;
    }
  }

  constructor(
    private api: ApiService,
    private auth: AuthService,
    private router: Router,
    private route: ActivatedRoute
  ) {}

  ngOnInit() {
    this.currentUser = this.auth.getCurrentUser() || {};
    const role = this.currentUser.Role || this.currentUser.role;
    this.isSuperAdmin = (role === 'Admin' || role === 'SuperAdmin' || role === 'DHE' || !!this.currentUser?.isAdmin);
    this.loadStats();
    this.loadUsers();
    this.loadDocTypes();
    this.loadFileCategories();
    this.loadFileSubCategories();
    const hasAdminAccess = !!this.currentUser?.isAdmin || role === 'SuperAdmin' || role === 'Admin' || role === 'DHE' || role === 'School Admin';
    if (hasAdminAccess) {
      this.loadSchools();
      this.loadRoles();
    }

    // Determine active section based on the URL path
    this.updateActiveSectionFromUrl(this.router.url);

    // Subscribe to router events to handle URL updates dynamically
    this.router.events.subscribe(() => {
      this.updateActiveSectionFromUrl(this.router.url);
    });
  }

  evaluatePermissions() {
    const role = this.currentUser?.Role || this.currentUser?.role;
    const userRoleRec = this.dbRoles.find(r => r.RoleName === role);
    this.isSuperAdmin = (role === 'SuperAdmin' || !userRoleRec?.ParentRoleID || userRoleRec?.ParentRoleName === 'SuperAdmin');
  }

  updateActiveSectionFromUrl(url: string) {
    const path = url.split('?')[0]; // strip query parameters
    const role = this.currentUser?.Role || this.currentUser?.role;
    const userRoleRec = this.dbRoles.find(r => r.RoleName === role);
    this.isSuperAdmin = (role === 'SuperAdmin' || !userRoleRec?.ParentRoleID || userRoleRec?.ParentRoleName === 'SuperAdmin');
    const isDHEOrAdmin = this.isSuperAdmin;

    if (path.endsWith('/users')) {
      this.activeSection = 'users';
    } else if (path.endsWith('/schools')) {
      const hasAdminAccess = !!this.currentUser?.isAdmin || role === 'SuperAdmin' || role === 'Admin' || role === 'DHE' || role === 'School Admin';
      if (hasAdminAccess) {
        this.activeSection = 'schools';
      } else {
        this.activeSection = 'overview';
        this.router.navigate(['/admin']);
      }
    } else if (path.endsWith('/doctypes')) {
      this.activeSection = 'doctypes';
    } else if (path.endsWith('/roles')) {
      const hasAdminAccess = !!this.currentUser?.isAdmin || role === 'SuperAdmin' || role === 'Admin' || role === 'DHE' || role === 'School Admin';
      if (hasAdminAccess) {
        this.activeSection = 'roles';
      } else {
        this.activeSection = 'overview';
        this.router.navigate(['/admin']);
      }
    } else {
      this.activeSection = 'overview';
    }
    this.clearMessages();
  }

  setSection(section: string) {
    this.activeSection = section;
    this.clearMessages();
  }

  clearMessages() {
    this.userError = ''; this.userSuccess = '';
    this.docTypeError = ''; this.docTypeSuccess = '';
    this.schoolError = ''; this.schoolSuccess = '';
    this.fileCategoryError = ''; this.fileCategorySuccess = '';
    this.fileSubCategoryError = ''; this.fileSubCategorySuccess = '';
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
    let defaultSchoolId = null;
    if (this.isSuperAdmin) {
      defaultSchoolId = this.schools[0]?.ID || null;
    } else {
      defaultSchoolId = this.currentUser.SchoolID || null;
    }
    this.userForm = {
      name: '',
      email: '',
      role: this.availableRoles[0] || 'vocational',
      password: '',
      class_section: '',
      subject: '',
      phone: '',
      school_id: defaultSchoolId
    };
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
      name: '', slug: '',
      workflow_stages: '[]', required_fields: '[]',
      sla_hours: 72, active: true,
      school_id: null   // always global — inherited by all child orgs
    };
    this.docTypeError = '';
    this.showDocTypeModal = true;
  }

  openEditDocType(dt: any) {
    this.editingDocType = dt;
    this.docTypeForm = {
      name: dt.Name,
      slug: dt.Slug,
      workflow_stages: dt.WorkflowStages || '[]',
      required_fields: dt.RequiredFields || '[]',
      sla_hours: dt.SlaHours,
      active: dt.Active,
      school_id: null   // always global — inherited by all child orgs
    };
    this.docTypeError = '';
    this.showDocTypeModal = true;
  }

  saveDocType() {
    this.docTypeError = '';
    if (!this.docTypeForm.name) {
      this.docTypeError = 'Receipt type name is required.';
      return;
    }

    // Ensure school_id is always null so the type is global and inherited by all orgs
    this.docTypeForm.school_id = null;

    if (!this.docTypeForm.slug) {
      this.docTypeForm.slug = this.docTypeForm.name.toLowerCase().replace(/\s+/g, '-');
    }
    const payload = { ...this.docTypeForm };
    if (this.editingDocType) {
      this.api.adminUpdateDocumentType(this.editingDocType.ID, payload).subscribe({
        next: () => {
          this.showDocTypeModal = false;
          this.docTypeSuccess = 'Receipt type updated.';
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
          this.docTypeSuccess = 'Receipt type created.';
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
        this.docTypeSuccess = 'Receipt type deleted.';
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
    this.api.getOrganizations().subscribe({
      next: (data) => {
        this.schools = data || []; // schools property serves as our organizations list
        this.loadingSchools = false;
      },
      error: () => { this.loadingSchools = false; }
    });
  }

  openCreateSchool() {
    this.editingSchool = null;
    let defaultParentId = null;
    const role = this.currentUser?.Role || this.currentUser?.role;
    if (role !== 'SuperAdmin' && this.currentUser && this.currentUser.SchoolID) {
      const myOrg = this.schools.find(s => s.TenantID === this.currentUser.SchoolID);
      if (myOrg) {
        defaultParentId = myOrg.ID;
      }
    }
    this.schoolForm = {
      organizationName: '',
      type: 'school',
      parentOrgId: defaultParentId,
      pointOfContactId: null,
      adminEmail: '',
      adminPassword: ''
    };
    this.schoolError = '';
    this.showSchoolModal = true;
  }

  openEditSchool(org: any) {
    this.editingSchool = org;
    this.schoolForm = {
      organizationName: org.OrganizationName,
      type: org.Type,
      parentOrgId: org.ParentOrgID,
      pointOfContactId: org.PointOfContactID,
      adminEmail: '',
      adminPassword: ''
    };
    this.schoolError = '';
    this.showSchoolModal = true;
  }

  saveSchool() {
    this.schoolError = '';
    if (this.editingSchool) {
      const payload = {
        organizationName: this.schoolForm.organizationName,
        type: this.schoolForm.type,
        parentOrgId: this.schoolForm.parentOrgId,
        pointOfContactId: this.schoolForm.pointOfContactId
      };
      this.api.updateOrganization(this.editingSchool.ID, payload).subscribe({
        next: () => {
          this.showSchoolModal = false;
          this.schoolSuccess = 'Organization updated successfully.';
          this.loadSchools();
          setTimeout(() => this.schoolSuccess = '', 3000);
        },
        error: (e) => this.schoolError = e.error?.error || 'Failed to update organization.'
      });
    } else {
      this.api.createOrganization(this.schoolForm).subscribe({
        next: () => {
          this.showSchoolModal = false;
          this.schoolSuccess = 'Organization created successfully.';
          this.loadSchools();
          setTimeout(() => this.schoolSuccess = '', 3000);
        },
        error: (e) => this.schoolError = e.error?.error || 'Failed to create organization.'
      });
    }
  }

  deleteOrganization(id: string) {
    if (confirm('Are you sure you want to delete this organization?')) {
      this.api.deleteOrganization(id).subscribe({
        next: () => {
          this.schoolSuccess = 'Organization deleted successfully.';
          this.loadSchools();
          setTimeout(() => this.schoolSuccess = '', 3000);
        },
        error: (e) => {
          alert(e.error?.error || 'Failed to delete organization.');
        }
      });
    }
  }

  // ── File Categories ────────────────────────────────────────────────────────

  loadFileCategories() {
    const stored = localStorage.getItem('file_categories');
    if (stored) {
      try {
        this.fileCategories = JSON.parse(stored);
      } catch (e) {
        this.initializeDefaultFileCategories();
      }
    } else {
      this.initializeDefaultFileCategories();
    }
    this.applyFileCategoryFilter();
  }

  initializeDefaultFileCategories() {
    this.fileCategories = [
      {ID: 'cat-admin', Name: 'Administration', Slug: 'administration', Active: true},
      {ID: 'cat-hr', Name: 'Human Resources', Slug: 'human-resources', Active: true},
      {ID: 'cat-finance', Name: 'Finance', Slug: 'finance', Active: true},
      {ID: 'cat-academic', Name: 'Academic', Slug: 'academic', Active: true},
      {ID: 'cat-infra', Name: 'Infrastructure', Slug: 'infrastructure', Active: true},
      {ID: 'cat-student', Name: 'Student Affairs', Slug: 'student-affairs', Active: true}
    ];
    localStorage.setItem('file_categories', JSON.stringify(this.fileCategories));
  }

  applyFileCategoryFilter() {
    const q = this.fileCategorySearch.toLowerCase().trim();
    this.filteredFileCategories = q
      ? this.fileCategories.filter(c => c.Name.toLowerCase().includes(q) || c.Slug.toLowerCase().includes(q))
      : [...this.fileCategories];
  }

  openCreateFileCategory() {
    this.editingFileCategory = null;
    this.fileCategoryForm = { name: '', slug: '', active: true };
    this.fileCategoryError = '';
    this.showFileCategoryModal = true;
  }

  openEditFileCategory(cat: any) {
    this.editingFileCategory = cat;
    this.fileCategoryForm = { name: cat.Name, slug: cat.Slug, active: cat.Active };
    this.fileCategoryError = '';
    this.showFileCategoryModal = true;
  }

  saveFileCategory() {
    this.fileCategoryError = '';
    const name = this.fileCategoryForm.name.trim();
    if (!name) {
      this.fileCategoryError = 'Category name is required.';
      return;
    }
    const slug = this.fileCategoryForm.slug.trim() || name.toLowerCase().replace(/\s+/g, '-');

    if (this.editingFileCategory) {
      const idx = this.fileCategories.findIndex(c => c.ID === this.editingFileCategory.ID);
      if (idx !== -1) {
        this.fileCategories[idx] = {
          ...this.fileCategories[idx],
          Name: name,
          Slug: slug,
          Active: this.fileCategoryForm.active
        };
      }
      this.fileCategorySuccess = 'File category updated successfully.';
    } else {
      const newCat = {
        ID: 'cat-' + Date.now().toString(36),
        Name: name,
        Slug: slug,
        Active: this.fileCategoryForm.active
      };
      this.fileCategories.push(newCat);
      this.fileCategorySuccess = 'File category created successfully.';
    }

    localStorage.setItem('file_categories', JSON.stringify(this.fileCategories));
    this.showFileCategoryModal = false;
    this.applyFileCategoryFilter();
    setTimeout(() => this.fileCategorySuccess = '', 3000);
  }

  confirmDeleteFileCategory(id: string) {
    this.deleteConfirmFileCategoryId = id;
  }

  deleteFileCategory(id: string) {
    this.fileCategories = this.fileCategories.filter(c => c.ID !== id);
    localStorage.setItem('file_categories', JSON.stringify(this.fileCategories));
    this.deleteConfirmFileCategoryId = '';
    this.fileCategorySuccess = 'File category deleted.';
    this.applyFileCategoryFilter();
    setTimeout(() => this.fileCategorySuccess = '', 3000);
  }

  // ── File Sub-categories ─────────────────────────────────────────────────────

  loadFileSubCategories() {
    const stored = localStorage.getItem('file_sub_categories');
    if (stored) {
      try {
        this.fileSubCategories = JSON.parse(stored);
      } catch (e) {
        this.initializeDefaultFileSubCategories();
      }
    } else {
      this.initializeDefaultFileSubCategories();
    }
    this.applyFileSubCategoryFilter();
  }

  initializeDefaultFileSubCategories() {
    this.fileSubCategories = [
      {ID: "sub-policy", Category: "Administration", Name: "Policy", Slug: "policy", Active: true},
      {ID: "sub-meetings", Category: "Administration", Name: "Meetings", Slug: "meetings", Active: true},
      {ID: "sub-audit-admin", Category: "Administration", Name: "Audit", Slug: "audit", Active: true},
      {ID: "sub-general", Category: "Administration", Name: "General", Slug: "general", Active: true},
      {ID: "sub-recruitment", Category: "Human Resources", Name: "Recruitment", Slug: "recruitment", Active: true},
      {ID: "sub-payroll", Category: "Human Resources", Name: "Payroll", Slug: "payroll", Active: true},
      {ID: "sub-grievance", Category: "Human Resources", Name: "Grievance", Slug: "grievance", Active: true},
      {ID: "sub-leave", Category: "Human Resources", Name: "Leave", Slug: "leave", Active: true},
      {ID: "sub-budget", Category: "Finance", Name: "Budget", Slug: "budget", Active: true},
      {ID: "sub-procure", Category: "Finance", Name: "Procurement", Slug: "procurement", Active: true},
      {ID: "sub-reimburse", Category: "Finance", Name: "Reimbursement", Slug: "reimbursement", Active: true},
      {ID: "sub-audit-fin", Category: "Finance", Name: "Audit", Slug: "audit", Active: true},
      {ID: "sub-curriculum", Category: "Academic", Name: "Curriculum", Slug: "curriculum", Active: true},
      {ID: "sub-exams", Category: "Academic", Name: "Exams", Slug: "exams", Active: true},
      {ID: "sub-admissions", Category: "Academic", Name: "Admissions", Slug: "admissions", Active: true},
      {ID: "sub-results", Category: "Academic", Name: "Results", Slug: "results", Active: true},
      {ID: "sub-maint", Category: "Infrastructure", Name: "Maintenance", Slug: "maintenance", Active: true},
      {ID: "sub-it", Category: "Infrastructure", Name: "IT Support", Slug: "it-support", Active: true},
      {ID: "sub-civil", Category: "Infrastructure", Name: "Civil Works", Slug: "civil-works", Active: true},
      {ID: "sub-complaints", Category: "Infrastructure", Name: "Complaints", Slug: "complaints", Active: true},
      {ID: "sub-disc", Category: "Student Affairs", Name: "Disciplinary", Slug: "disciplinary", Active: true},
      {ID: "sub-events", Category: "Student Affairs", Name: "Events", Slug: "events", Active: true},
      {ID: "sub-scholar", Category: "Student Affairs", Name: "Scholarships", "Slug": "scholarships", "Active": true},
      {ID: "sub-hostel", Category: "Student Affairs", Name: "Hostel", Slug: "hostel", Active: true}
    ];
    localStorage.setItem('file_sub_categories', JSON.stringify(this.fileSubCategories));
  }

  applyFileSubCategoryFilter() {
    const q = this.fileSubCategorySearch.toLowerCase().trim();
    this.filteredFileSubCategories = q
      ? this.fileSubCategories.filter(s => s.Name.toLowerCase().includes(q) || s.Category.toLowerCase().includes(q) || s.Slug.toLowerCase().includes(q))
      : [...this.fileSubCategories];
  }

  openCreateFileSubCategory() {
    this.editingFileSubCategory = null;
    this.fileSubCategoryForm = { category: this.fileCategories[0]?.Name || '', name: '', slug: '', active: true };
    this.fileSubCategoryError = '';
    this.showFileSubCategoryModal = true;
  }

  openEditFileSubCategory(sub: any) {
    this.editingFileSubCategory = sub;
    this.fileSubCategoryForm = { category: sub.Category, name: sub.Name, slug: sub.Slug, active: sub.Active };
    this.fileSubCategoryError = '';
    this.showFileSubCategoryModal = true;
  }

  saveFileSubCategory() {
    this.fileSubCategoryError = '';
    const name = this.fileSubCategoryForm.name.trim();
    const category = this.fileSubCategoryForm.category.trim();
    if (!name) {
      this.fileSubCategoryError = 'Sub-category name is required.';
      return;
    }
    if (!category) {
      this.fileSubCategoryError = 'Parent Category is required.';
      return;
    }
    const slug = this.fileSubCategoryForm.slug.trim() || name.toLowerCase().replace(/\s+/g, '-');

    if (this.editingFileSubCategory) {
      const idx = this.fileSubCategories.findIndex(s => s.ID === this.editingFileSubCategory.ID);
      if (idx !== -1) {
        this.fileSubCategories[idx] = {
          ...this.fileSubCategories[idx],
          Category: category,
          Name: name,
          Slug: slug,
          Active: this.fileSubCategoryForm.active
        };
      }
      this.fileSubCategorySuccess = 'File sub-category updated successfully.';
    } else {
      const newSub = {
        ID: 'sub-' + Date.now().toString(36),
        Category: category,
        Name: name,
        Slug: slug,
        Active: this.fileSubCategoryForm.active
      };
      this.fileSubCategories.push(newSub);
      this.fileSubCategorySuccess = 'File sub-category created successfully.';
    }

    localStorage.setItem('file_sub_categories', JSON.stringify(this.fileSubCategories));
    this.showFileSubCategoryModal = false;
    this.applyFileSubCategoryFilter();
    setTimeout(() => this.fileSubCategorySuccess = '', 3000);
  }

  confirmDeleteFileSubCategory(id: string) {
    this.deleteConfirmFileSubCategoryId = id;
  }

  deleteFileSubCategory(id: string) {
    this.fileSubCategories = this.fileSubCategories.filter(s => s.ID !== id);
    localStorage.setItem('file_sub_categories', JSON.stringify(this.fileSubCategories));
    this.deleteConfirmFileSubCategoryId = '';
    this.fileSubCategorySuccess = 'File sub-category deleted.';
    this.applyFileSubCategoryFilter();
    setTimeout(() => this.fileSubCategorySuccess = '', 3000);
  }

  // ── Helpers ────────────────────────────────────────────────────────────────

  countByRole(role: string): number {
    return this.users.filter(u => u.Role === role).length;
  }

  formatDate(d: string): string {
    if (!d) return '—';
    return new Date(d).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
  }

  // ── Role Management CRUD ───────────────────────────────────────────────────

  loadRoles() {
    this.loadingRoles = true;
    this.roleError = '';
    this.api.getRoles().subscribe({
      next: (res: any) => {
        this.dbRoles = res || [];
        this.applyRoleFilter();
        this.evaluatePermissions();
        this.loadingRoles = false;
      },
      error: (err: any) => {
        this.roleError = err.error?.error || 'Failed to load roles';
        this.loadingRoles = false;
      }
    });
  }

  applyRoleFilter() {
    const q = this.roleSearch.toLowerCase().trim();
    if (!q) {
      this.filteredDbRoles = [...this.dbRoles];
    } else {
      this.filteredDbRoles = this.dbRoles.filter(
        r => r.RoleName.toLowerCase().includes(q) || (r.ParentRoleName && r.ParentRoleName.toLowerCase().includes(q))
      );
    }
  }

  openCreateRole() {
    this.editingRole = null;
    this.roleForm = { roleName: '', isAdminAccess: false, parentRole: null };
    this.roleError = '';
    this.roleSuccess = '';
    this.showRoleModal = true;
  }

  openEditRole(role: any) {
    this.editingRole = role;
    this.roleForm = {
      roleName: role.RoleName,
      isAdminAccess: role.IsAdminAccess,
      parentRole: role.ParentRoleID || null
    };
    this.roleError = '';
    this.roleSuccess = '';
    this.showRoleModal = true;
  }

  saveRole() {
    if (!this.roleForm.roleName.trim()) {
      this.roleError = 'Role name is required';
      return;
    }

    const payload = {
      roleName: this.roleForm.roleName,
      isAdminAccess: this.roleForm.isAdminAccess,
      parentRole: this.roleForm.parentRole || null,
      tenantId: this.editingRole?.TenantID || null
    };

    this.roleError = '';
    this.roleSuccess = '';

    if (this.editingRole) {
      this.api.updateRole(this.editingRole.ID, payload).subscribe({
        next: () => {
          this.roleSuccess = 'Role updated successfully';
          this.showRoleModal = false;
          this.loadRoles();
          setTimeout(() => this.roleSuccess = '', 3000);
        },
        error: (err: any) => {
          this.roleError = err.error?.error || 'Failed to update role';
        }
      });
    } else {
      this.api.createRole(payload).subscribe({
        next: () => {
          this.roleSuccess = 'Role created successfully';
          this.showRoleModal = false;
          this.loadRoles();
          setTimeout(() => this.roleSuccess = '', 3000);
        },
        error: (err: any) => {
          this.roleError = err.error?.error || 'Failed to create role';
        }
      });
    }
  }

  confirmDeleteRole(id: string) {
    this.deleteConfirmRoleId = id;
  }

  deleteRole(id: string) {
    this.roleError = '';
    this.roleSuccess = '';
    this.api.deleteRole(id).subscribe({
      next: () => {
        this.roleSuccess = 'Role deleted successfully';
        this.deleteConfirmRoleId = '';
        this.loadRoles();
        setTimeout(() => this.roleSuccess = '', 3000);
      },
      error: (err: any) => {
        this.roleError = err.error?.error || 'Failed to delete role';
        this.deleteConfirmRoleId = '';
        setTimeout(() => this.roleError = '', 4000);
      }
    });
  }
}
