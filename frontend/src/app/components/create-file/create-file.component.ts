import { Component, OnInit } from '@angular/core';
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
export class CreateFileComponent implements OnInit {
  title: string = '';
  description: string = '';
  category: string = '';
  subCategory: string = '';
  priority: string = 'Normal';
  error: string = '';
  loading: boolean = false;

  categories: string[] = [];
  subCategories: Record<string, string[]> = {};
  availableSubCategories: string[] = [];

  constructor(
    private api: ApiService,
    private auth: AuthService,
    private router: Router
  ) {}

  ngOnInit() {
    this.loadCategoriesFromStorage();
  }

  loadCategoriesFromStorage() {
    // Categories
    const storedCats = localStorage.getItem('file_categories');
    if (storedCats) {
      try {
        const parsed = JSON.parse(storedCats);
        this.categories = parsed.filter((c: any) => c.Active !== false).map((c: any) => c.Name);
      } catch (e) {
        this.categories = ['Administration', 'Human Resources', 'Finance', 'Academic', 'Infrastructure', 'Student Affairs'];
      }
    } else {
      this.categories = ['Administration', 'Human Resources', 'Finance', 'Academic', 'Infrastructure', 'Student Affairs'];
    }

    // Sub-categories
    const storedSubCats = localStorage.getItem('file_sub_categories');
    if (storedSubCats) {
      try {
        const parsed = JSON.parse(storedSubCats);
        const record: Record<string, string[]> = {};
        parsed.forEach((sub: any) => {
          if (sub.Active !== false) {
            if (!record[sub.Category]) {
              record[sub.Category] = [];
            }
            record[sub.Category].push(sub.Name);
          }
        });
        this.subCategories = record;
      } catch (e) {
        this.loadDefaultSubCategories();
      }
    } else {
      this.loadDefaultSubCategories();
    }
  }

  loadDefaultSubCategories() {
    this.subCategories = {
      'Administration': ['Policy', 'Meetings', 'Audit', 'General'],
      'Human Resources': ['Recruitment', 'Payroll', 'Grievance', 'Leave'],
      'Finance': ['Budget', 'Procurement', 'Reimbursement', 'Audit'],
      'Academic': ['Curriculum', 'Exams', 'Admissions', 'Results'],
      'Infrastructure': ['Maintenance', 'IT Support', 'Civil Works', 'Complaints'],
      'Student Affairs': ['Disciplinary', 'Events', 'Scholarships', 'Hostel']
    };
  }

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
    this.api.createFile(titleTrimmed, this.description, this.category, this.subCategory, this.priority).subscribe({
      next: (res: any) => {
        this.loading = false;
        const fileId = res.ID || res.id;
        this.router.navigate(['/details', fileId], { queryParams: { type: 'file' } });
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
