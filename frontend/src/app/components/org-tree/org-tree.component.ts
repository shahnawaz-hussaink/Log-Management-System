import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';

export interface OrgNode {
  id: string;
  name: string;
  type: string;
  activeState: string;
  parentOrgID: string | null;
  pointOfContact: any | null;
  tenantID: string | null;
  children: OrgNode[];
  school: any | null;
  expanded: boolean;
}

@Component({
  selector: 'app-org-tree',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './org-tree.component.html',
  styleUrls: ['./org-tree.component.css'],
})
export class OrgTreeComponent implements OnInit {
  loading = true;
  error = '';

  allOrgs: any[] = [];
  allSchools: any[] = [];
  rootNodes: OrgNode[] = [];

  totalOrgs = 0;
  totalSchools = 0;

  constructor(
    private api: ApiService,
    private auth: AuthService,
    public router: Router
  ) {}

  ngOnInit() {
    const user = this.auth.getCurrentUser();
    if (!user || (user.Role !== 'SuperAdmin' && user.role !== 'SuperAdmin')) {
      this.router.navigate(['/dashboard']);
      return;
    }
    this.loadAll();
  }

  loadAll() {
    this.loading = true;
    let orgsLoaded = false;
    let schoolsLoaded = false;

    const tryBuild = () => {
      if (orgsLoaded && schoolsLoaded) {
        this.buildTree();
        this.loading = false;
      }
    };

    this.api.getOrganizations().subscribe({
      next: (res) => {
        this.allOrgs = res || [];
        this.totalOrgs = this.allOrgs.length;
        orgsLoaded = true;
        tryBuild();
      },
      error: () => { orgsLoaded = true; tryBuild(); }
    });

    this.api.getAdminSchools().subscribe({
      next: (res) => {
        this.allSchools = res || [];
        this.totalSchools = this.allSchools.length;
        schoolsLoaded = true;
        tryBuild();
      },
      error: () => { schoolsLoaded = true; tryBuild(); }
    });
  }

  buildTree() {
    const nodeMap = new Map<string, OrgNode>();

    // Create a node for each organization
    for (const org of this.allOrgs) {
      const school = this.allSchools.find(
        (s: any) => s.ID === org.TenantID || s.ID === org.ID
      ) || null;

      nodeMap.set(org.ID, {
        id: org.ID,
        name: org.OrganizationName,
        type: org.Type,
        activeState: org.ActiveState,
        parentOrgID: org.ParentOrgID || null,
        pointOfContact: org.PointOfContact || null,
        tenantID: org.TenantID || null,
        school: school,
        children: [],
        expanded: true, // Default to expanded
      });
    }

    // Wire up parent-child relationships
    const roots: OrgNode[] = [];
    for (const node of nodeMap.values()) {
      if (node.parentOrgID && nodeMap.has(node.parentOrgID)) {
        nodeMap.get(node.parentOrgID)!.children.push(node);
      } else {
        roots.push(node);
      }
    }

    this.rootNodes = roots;
  }

  toggleExpanded(node: OrgNode, event: Event) {
    event.stopPropagation();
    node.expanded = !node.expanded;
  }

  expandAll() {
    this.setExpanded(this.rootNodes, true);
  }

  collapseAll() {
    this.setExpanded(this.rootNodes, false);
  }

  setExpanded(nodes: OrgNode[], value: boolean) {
    for (const n of nodes) {
      n.expanded = value;
      this.setExpanded(n.children, value);
    }
  }

  countDescendants(node: OrgNode): number {
    let count = node.children.length;
    for (const c of node.children) count += this.countDescendants(c);
    return count;
  }
  
  trackById(index: number, item: any): string {
    return item.id;
  }
}