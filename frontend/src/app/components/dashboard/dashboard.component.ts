import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
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
  currentUser: any = null;
  searchText: string = '';

  constructor(private api: ApiService, private auth: AuthService, private router: Router) {}

  ngOnInit() {
    this.currentUser = this.auth.getCurrentUser();
    if (!this.currentUser) {
      this.router.navigate(['/login']);
      return;
    }
    this.loadDocuments();
  }

  loadDocuments() {
    this.api.getDocuments(this.currentUser.ID, this.searchText).subscribe({
      next: (docs) => {
        this.documents = docs;
      }
    });
  }

  onSearchChange() {
    this.loadDocuments();
  }

  goToUpload() {
    this.router.navigate(['/upload']);
  }

  goToDetails(id: string) {
    this.router.navigate(['/details', id]);
  }
}
