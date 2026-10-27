/** Capabilities required by enrollment and recovery, independent of Firebase SDK types. */
export interface StudentIdentity {
  readonly uid: string;
  readonly isAnonymous: boolean;
  getIdToken(forceRefresh?: boolean): Promise<string>;
  delete(): Promise<void>;
}
