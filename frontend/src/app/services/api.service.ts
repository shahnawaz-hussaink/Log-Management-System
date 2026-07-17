import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class ApiService {
  private apiUrl = 'http://localhost:8080/api';

  public searchSubject = new BehaviorSubject<string>('');
  public activeTabSubject = new BehaviorSubject<string>('pending_me');

  constructor(private http: HttpClient) {}

  getUsers() {
    return this.http.get<any[]>(`${this.apiUrl}/users`);
  }

  getDocumentTypes() {
    return this.http.get<any[]>(`${this.apiUrl}/document-types`);
  }

  getDocuments(userId: string, search?: string) {
    let url = `${this.apiUrl}/documents?user_id=${userId}`;
    if (search) {
      url += `&search=${encodeURIComponent(search)}`;
    }
    return this.http.get<any[]>(url);
  }

  getDocumentDetails(id: string) {
    return this.http.get<any>(`${this.apiUrl}/documents/${id}`);
  }

  getSubmissions(id: string) {
    return this.http.get<any[]>(`${this.apiUrl}/documents/${id}/submissions`);
  }

  uploadDocument(formData: FormData) {
    return this.http.post<any>(`${this.apiUrl}/documents`, formData);
  }

  replaceDocument(id: string, formData: FormData) {
    return this.http.put<any>(
      `${this.apiUrl}/documents/${id}/replace`,
      formData,
    );
  }

  submitAction(id: string, actionData: any) {
    return this.http.post<any>(
      `${this.apiUrl}/documents/${id}/action`,
      actionData,
    );
  }

  login(email: string, password?: string) {
    return this.http.post<any>(`${this.apiUrl}/auth/login`, {
      email,
      password,
    });
  }

  signup(name: string, email: string, password?: string) {
    return this.http.post<any>(`${this.apiUrl}/auth/signup`, {
      name,
      email,
      password,
    });
  }

  appendNote(id: string, note: string) {
    return this.http.post<any>(`${this.apiUrl}/documents/${id}/notes`, {
      note,
    });
  }

  saveDraft(id: string, draft: string) {
    return this.http.put<any>(`${this.apiUrl}/documents/${id}/draft`, {
      draft,
    });
  }

  addAttachment(id: string, file: File) {
    const formData = new FormData();
    formData.append('file', file);
    return this.http.post<any>(
      `${this.apiUrl}/documents/${id}/attachments`,
      formData,
    );
  }

  getNotifications() {
    return this.http.get<any[]>(`${this.apiUrl}/notifications`);
  }

  getReports() {
    return this.http.get<any>(`${this.apiUrl}/reports`);
  }

  getMyHistory() {
    return this.http.get<any[]>(`${this.apiUrl}/my-history`);
  }

  sendManualEmail(to: string, subject: string, body: string) {
    return this.http.post<any>(`${this.apiUrl}/send-email`, {
      to,
      subject,
      body,
    });
  }

  // ── Admin API ──────────────────────────────────────────────────────────────

  getAdminStats() {
    return this.http.get<any>(`${this.apiUrl}/admin/stats`);
  }

  getAdminUsers() {
    return this.http.get<any[]>(`${this.apiUrl}/admin/users`);
  }

  adminCreateUser(data: any) {
    return this.http.post<any>(`${this.apiUrl}/admin/users`, data);
  }

  adminUpdateUser(id: string, data: any) {
    return this.http.put<any>(`${this.apiUrl}/admin/users/${id}`, data);
  }

  adminDeleteUser(id: string) {
    return this.http.delete<any>(`${this.apiUrl}/admin/users/${id}`);
  }

  getAdminDocumentTypes() {
    return this.http.get<any[]>(`${this.apiUrl}/admin/document-types`);
  }

  adminCreateDocumentType(data: any) {
    return this.http.post<any>(`${this.apiUrl}/admin/document-types`, data);
  }

  adminUpdateDocumentType(id: string, data: any) {
    return this.http.put<any>(
      `${this.apiUrl}/admin/document-types/${id}`,
      data,
    );
  }

  adminDeleteDocumentType(id: string) {
    return this.http.delete<any>(`${this.apiUrl}/admin/document-types/${id}`);
  }

  getAdminSchools() {
    return this.http.get<any[]>(`${this.apiUrl}/admin/schools`);
  }

  adminUpdateSchool(id: string, data: any) {
    return this.http.put<any>(`${this.apiUrl}/admin/schools/${id}`, data);
  }

  recallDocument(docId: string) {
    return this.http.post<any>(`${this.apiUrl}/documents/${docId}/recall`, {});
  }
}
