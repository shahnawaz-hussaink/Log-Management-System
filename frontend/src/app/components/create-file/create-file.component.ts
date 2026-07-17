import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-create-file',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './create-file.component.html',
  styleUrls: ['./create-file.component.css']
})
export class CreateFileComponent {
  title: string = '';
  description: string = '';
  category: string = '';
  subCategory: string = '';
  error: string = '';
  loading: boolean = false;

  categories = [
    'Administration',
    'Human Resources',
    'Finance',
    'Academic',
    'Infrastructure',
    'Student Affairs'
  ];

  subCategories: Record<string, string[]> = {
    'Administration': ['Policy', 'Meetings', 'Audit', 'General'],
    'Human Resources': ['Recruitment', 'Payroll', 'Grievance', 'Leave'],
    'Finance': ['Budget', 'Procurement', 'Reimbursement', 'Audit'],
    'Academic': ['Curriculum', 'Exams', 'Admissions', 'Results'],
    'Infrastructure': ['Maintenance', 'IT Support', 'Civil Works', 'Complaints'],
    'Student Affairs': ['Disciplinary', 'Events', 'Scholarships', 'Hostel']
  };

  availableSubCategories: string[] = [];

  constructor(
    private api: ApiService,
    private auth: AuthService,
    private router: Router
  ) {}

  onCategoryChange() {
    this.availableSubCategories = this.subCategories[this.category] || [];
    this.subCategory = ''; // Reset sub-category when category changes
  }

  createFile() {
    this.error = '';
    const titleTrimmed = this.title.trim();
    if (!titleTrimmed) {
      this.error = 'File Title is required.';
      return;
    }
    if (!this.category) {
      this.error = 'Category is required.';
      return;
    }
    if (!this.subCategory) {
      this.error = 'Sub-Category is required.';
      return;
    }

    this.loading = true;
    this.api.createFile(titleTrimmed, this.description, this.category, this.subCategory).subscribe({
      next: (res: any) => {
        this.loading = false;
        this.router.navigate(['/dashboard']);
      },
      error: (err: any) => {
        this.loading = false;
        this.error = err.error?.error || 'Failed to create file.';
      }
    });
  }

  cancel() {
    this.router.navigate(['/dashboard']);
  }
}
