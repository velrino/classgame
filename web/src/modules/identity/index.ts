export { TeacherAuthProvider } from "./adapters/context/teacher-auth-context";
export { identityApi } from "./adapters/http";
export {
  currentStudentIdentity,
  ensureAnonymousStudent,
  isIdentityConfigured,
  observeStudentIdentity,
  signOutStudentIdentity,
} from "./adapters/student-identity";
export type { StudentIdentity } from "./adapters/types/student-identity";
export { useTeacherAuth } from "./hooks/use-teacher-auth";
export { LoginPage } from "./ui/login-page";
export { TeacherDashboardGate } from "./ui/teacher-dashboard-gate";
