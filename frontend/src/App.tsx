import { BrowserRouter, Route, Routes } from "react-router-dom";
import { RoleProvider } from "./context/RoleContext";
import { Layout } from "./components/Layout";
import { Intake } from "./pages/Intake";
import { Prospects } from "./pages/Prospects";
import { ReviewQueue } from "./pages/ReviewQueue";
import { CompanyMaster } from "./pages/CompanyMaster";
import { UW } from "./pages/UW";
import { Monitoring } from "./pages/Monitoring";
import { AuditLogPage } from "./pages/AuditLog";

export default function App() {
  return (
    <RoleProvider>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<Intake />} />
            <Route path="prospects" element={<Prospects />} />
            <Route path="review-queue" element={<ReviewQueue />} />
            <Route path="companies" element={<CompanyMaster />} />
            <Route path="uw" element={<UW />} />
            <Route path="monitoring" element={<Monitoring />} />
            <Route path="audit-log" element={<AuditLogPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </RoleProvider>
  );
}
