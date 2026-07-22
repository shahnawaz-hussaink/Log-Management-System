import { Routes } from '@angular/router';
import { LoginComponent } from './components/login/login.component';
import { RegisterComponent } from './components/register/register.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { UploadComponent } from './components/upload/upload.component';
import { CreateFileComponent } from './components/create-file/create-file.component';
import { DetailsComponent } from './components/details/details.component';
import { AdminComponent } from './components/admin/admin.component';
import { HistoryComponent } from './components/history/history.component';
import { ProfileComponent } from './components/profile/profile.component';
import { NotFoundComponent } from './components/not-found/not-found.component';
import { OrgTreeComponent } from './components/org-tree/org-tree.component';
import { authGuard } from './auth.guard';
import { guestGuard } from './guest.guard';
import { adminGuard } from './admin.guard';

export const routes: Routes = [
  { path: '',           redirectTo: '/login', pathMatch: 'full' },
  { path: 'login',      component: LoginComponent,     canActivate: [guestGuard] },
  { path: 'register',   component: RegisterComponent,  canActivate: [guestGuard] },
  { path: 'dashboard',  component: DashboardComponent, canActivate: [authGuard] },
  { path: 'receipt',    component: UploadComponent,    canActivate: [authGuard] },
  { path: 'create-file',component: CreateFileComponent,canActivate: [authGuard] },
  { path: 'history',    component: HistoryComponent,   canActivate: [authGuard] },
  { path: 'details/:id',component: DetailsComponent,   canActivate: [authGuard] },
  { path: 'profile',    component: ProfileComponent,   canActivate: [authGuard] },
  // Admin route and sub-routes
  { path: 'admin',          component: AdminComponent,     canActivate: [authGuard, adminGuard] },
  { path: 'admin/users',    component: AdminComponent,     canActivate: [authGuard, adminGuard] },
  { path: 'admin/schools',  component: AdminComponent,     canActivate: [authGuard, adminGuard] },
  { path: 'admin/doctypes', component: AdminComponent,     canActivate: [authGuard, adminGuard] },
  { path: 'admin/roles',    component: AdminComponent,     canActivate: [authGuard, adminGuard] },
  { path: 'admin/org-tree', component: OrgTreeComponent,  canActivate: [authGuard, adminGuard] },
  // Wildcard 404 Route
  { path: '**',         component: NotFoundComponent }
];
