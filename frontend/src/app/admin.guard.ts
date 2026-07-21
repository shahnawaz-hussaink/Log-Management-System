import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { AuthService } from './services/auth.service';

// Protects /admin — allows "Admin" role only
export const adminGuard = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const user = auth.getCurrentUser();
  console.log('adminGuard: currentUser is', user);

  if (user) {
    const role = user.Role || user.role;
    console.log('adminGuard: detected role is', role);
    if (user.isAdmin || role === 'Admin' || role === 'SuperAdmin' || role === 'DHE' || role === 'School Admin') {
      return true;
    }
  }

  console.warn('adminGuard: access denied, redirecting to /dashboard');
  router.navigate(['/dashboard']);
  return false;
};

