import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { Router } from '@angular/router';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.css']
})
export class LoginComponent {
  email: string = 'alice@office.com';
  error: string = '';

  constructor(private api: ApiService, private auth: AuthService, private router: Router) {}

  login() {
    this.api.login(this.email).subscribe({
      next: (user) => {
        this.auth.setCurrentUser(user);
        this.router.navigate(['/dashboard']);
      },
      error: () => {
        this.error = 'Invalid email or user not found. (Hint: try alice@office.com, bob@office.com, or charlie@office.com)';
      }
    });
  }
}
