import { Route, Routes } from "react-router-dom";
import FoundationPage from "@/pages/FoundationPage";

export default function App() {
  return (
    <Routes>
      <Route path="*" element={<FoundationPage />} />
    </Routes>
  );
}
