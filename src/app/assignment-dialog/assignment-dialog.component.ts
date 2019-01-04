import { Component, OnInit, Inject } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material';
import { Objective } from '../objective';

export class PersonAssignmentData {
  constructor(
    public username: string,
    public available: number,
    public assign: number) {}
}

export interface AssignmentDialogData {
  objective: Objective;
  people: PersonAssignmentData[];
  unit: string;
  columns: string[];
}

@Component({
  selector: 'app-assignment-dialog',
  templateUrl: './assignment-dialog.component.html',
  styleUrls: ['./assignment-dialog.component.css']
})
export class AssignmentDialogComponent implements OnInit {
  constructor(
    public dialogRef: MatDialogRef<AssignmentDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: AssignmentDialogData
  ) {}

  ngOnInit(): void {}

  onCancel(): void {
    this.dialogRef.close();
  }

  /**
   * Amount of the estimated time for the objective which isn't yet assigned
   */
  unassignedTime(): number {
    let assigned = this.data.people.map(d => d.assign).reduce((sum, current) => sum + current, 0);
    return this.data.objective.resourceEstimate - assigned;
  }

  assignNone(row: PersonAssignmentData): void {
    row.assign = 0;
  }

  assignAll(row: PersonAssignmentData): void {
    row.assign = Math.min(row.available, this.data.objective.resourceEstimate);
  }

  assignRemaining(row: PersonAssignmentData): void {
    row.assign = Math.min(row.available, this.unassignedTime() + row.assign);
  }

  assignMore(row: PersonAssignmentData): void {
    row.assign = Math.min(row.available, row.assign + 1);
  }

  assignLess(row: PersonAssignmentData): void {
    row.assign = Math.max(0, row.assign - 1);
  }

  isFullyCommitted(row: PersonAssignmentData): boolean {
    return row.assign >= row.available;
  }

  isFullyUncommitted(row: PersonAssignmentData): boolean {
    return row.assign <= 0;
  }

  isDataValid(): boolean {
    return this.unassignedTime() >= 0;
  }
}