import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-profile',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './profile.component.html',
  styleUrls: ['./profile.component.css']
})
export class ProfileComponent implements OnInit {
  currentUser: any = null;
  private apiUrl = 'http://localhost:8080/api';

  // Avatar
  avatarInitials: string = '';
  avatarColor: string = '#3b82f6';
  avatarPreview: string | null = null;
  avatarFile: File | null = null;

  // Password reset
  currentPassword: string = '';
  newPassword: string = '';
  confirmPassword: string = '';
  showCurrentPassword: boolean = false;
  showNewPassword: boolean = false;
  showConfirmPassword: boolean = false;

  // Phone
  newPhone: string = '';

  // Status flags
  savingPassword: boolean = false;
  savingPhone: boolean = false;
  passwordSuccess: string = '';
  passwordError: string = '';
  phoneSuccess: string = '';
  phoneError: string = '';

  private avatarColors = [
    '#3b82f6', '#8b5cf6', '#06b6d4', '#10b981', '#f59e0b', '#ef4444'
  ];

  constructor(
    public router: Router,
    private auth: AuthService,
    private http: HttpClient
  ) {}

  ngOnInit(): void {
    this.auth.currentUser$.subscribe(user => {
      if (!user) {
        this.router.navigate(['/login']);
        return;
      }
      this.currentUser = user;
      this.newPhone = user.Phone || user.phone || '';
      const name: string = user.Name || user.name || '';
      const words = name.trim().split(' ').filter((w: string) => w.length > 0);
      this.avatarInitials = words.length >= 2
        ? (words[0][0] + words[words.length - 1][0]).toUpperCase()
        : name.slice(0, 2).toUpperCase();
      const colorIndex = name.charCodeAt(0) % this.avatarColors.length;
      this.avatarColor = this.avatarColors[colorIndex];
    });
  }

  onAvatarFileChange(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    const file = input.files[0];
    if (!file.type.startsWith('image/')) {
      return;
    }
    this.avatarFile = file;
    const reader = new FileReader();
    reader.onload = (e) => {
      this.avatarPreview = e.target?.result as string;
    };
    reader.readAsDataURL(file);
  }

  resetPassword(): void {
    this.passwordError = '';
    this.passwordSuccess = '';

    if (!this.currentPassword || !this.newPassword || !this.confirmPassword) {
      this.passwordError = 'All password fields are required.';
      return;
    }
    if (this.newPassword !== this.confirmPassword) {
      this.passwordError = 'New password and confirm password do not match.';
      return;
    }
    if (this.newPassword.length < 8) {
      this.passwordError = 'New password must be at least 8 characters.';
      return;
    }

    this.savingPassword = true;
    const userId = this.currentUser?.ID || this.currentUser?.id;
    this.http.put(`${this.apiUrl}/profile/password`, {
      current_password: this.currentPassword,
      new_password: this.newPassword
    }).subscribe({
      next: () => {
        this.savingPassword = false;
        this.passwordSuccess = 'Password updated successfully.';
        this.currentPassword = '';
        this.newPassword = '';
        this.confirmPassword = '';
        setTimeout(() => this.passwordSuccess = '', 4000);
      },
      error: (err) => {
        this.savingPassword = false;
        this.passwordError = err.error?.error || 'Failed to update password.';
      }
    });
  }

  updatePhone(): void {
    this.phoneError = '';
    this.phoneSuccess = '';

    if (!this.newPhone.trim()) {
      this.phoneError = 'Phone number cannot be empty.';
      return;
    }

    this.savingPhone = true;
    this.http.put(`${this.apiUrl}/profile/phone`, {
      phone: this.newPhone.trim()
    }).subscribe({
      next: (updatedUser: any) => {
        this.savingPhone = false;
        this.phoneSuccess = 'Mobile number updated successfully.';
        // Update the local user store
        const user = { ...this.currentUser, Phone: this.newPhone.trim(), phone: this.newPhone.trim() };
        this.auth.setCurrentUser(user, this.auth.getToken()!);
        setTimeout(() => this.phoneSuccess = '', 4000);
      },
      error: (err) => {
        this.savingPhone = false;
        this.phoneError = err.error?.error || 'Failed to update mobile number.';
      }
    });
  }
}
