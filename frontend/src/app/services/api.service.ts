import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private apiUrl = 'http://localhost:8080/api';

  constructor(private http: HttpClient) {}

  getUsers() {
    return this.http.get<any[]>(`${this.apiUrl}/users`);
  }

  getDocuments(userId: string) {
    return this.http.get<any[]>(`${this.apiUrl}/documents?user_id=${userId}`);
  }

  getDocumentDetails(id: string) {
    return this.http.get<any>(`${this.apiUrl}/documents/${id}`);
  }

  uploadDocument(formData: FormData) {
    return this.http.post<any>(`${this.apiUrl}/documents`, formData);
  }

  replaceDocument(id: string, formData: FormData) {
    return this.http.put<any>(`${this.apiUrl}/documents/${id}/replace`, formData);
  }

  submitAction(id: string, actionData: any) {
    return this.http.post<any>(`${this.apiUrl}/documents/${id}/action`, actionData);
  }

  login(email: string) {
    return this.http.post<any>(`${this.apiUrl}/auth/login`, { email });
  }
}
