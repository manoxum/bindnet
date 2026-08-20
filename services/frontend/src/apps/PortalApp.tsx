import { Navigate, Route, Routes } from "react-router-dom";
import { PortalPage } from "@/pages/Portal";
import { PortalCaPage } from "@/pages/PortalCa";

export default function PortalApp() {
  return (
    <Routes>
      <Route index element={<PortalPage />} />
      <Route path="/ca" element={<PortalCaPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
