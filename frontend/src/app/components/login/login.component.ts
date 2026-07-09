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
  password: string = 'password';
  error: string = '';

  constructor(private api: ApiService, private auth: AuthService, private router: Router) {}

  login() {
    this.api.login(this.email, this.password).subscribe({
      next: (res) => {
        this.auth.setCurrentUser(res.user, res.token);
        this.router.navigate(['/dashboard']);
      },
      error: () => {
        this.error = 'Invalid email/password or user not found. (Hint: default password is "password")';
      }
    });
  }
}
