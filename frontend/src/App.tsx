import { Navigate, Route, Routes } from "react-router-dom";
import { useSession } from "@/session";
import FoundationPage from "@/pages/FoundationPage";
import CheckEmail from "@/pages/account/CheckEmail";
import ConfirmEmail from "@/pages/account/ConfirmEmail";
import CreateAccount from "@/pages/account/CreateAccount";
import FindOrganization from "@/pages/account/FindOrganization";
import ForgotPassword from "@/pages/account/ForgotPassword";
import Home from "@/pages/account/Home";
import LoggedOut from "@/pages/account/LoggedOut";
import Login from "@/pages/account/Login";
import Logout from "@/pages/account/Logout";
import ResetPassword from "@/pages/account/ResetPassword";
import UnknownAddress from "@/pages/account/UnknownAddress";

/*
  What the page shows depends on the address (4A.5): the general address
  shows A1; an organization's address shows that organization's screens; any
  other address names no organization. /status is Phase 1's health check.
*/
export default function App() {
  const { address, addressError, session, restoring } = useSession();

  if (addressError) {
    return (
      <Routes>
        <Route path="/status" element={<FoundationPage />} />
        <Route path="*" element={<UnknownAddress unreachable />} />
      </Routes>
    );
  }
  if (!address || restoring) return null;

  if (address.kind === "general") {
    return (
      <Routes>
        <Route path="/status" element={<FoundationPage />} />
        <Route path="*" element={<FindOrganization />} />
      </Routes>
    );
  }
  if (address.kind === "unknown") {
    return (
      <Routes>
        <Route path="/status" element={<FoundationPage />} />
        <Route path="*" element={<UnknownAddress />} />
      </Routes>
    );
  }

  const signedIn = !!session;
  return (
    <Routes>
      <Route path="/status" element={<FoundationPage />} />
      <Route path="/" element={signedIn ? <Home /> : <Navigate to="/login" replace />} />
      <Route path="/login" element={signedIn ? <Navigate to="/" replace /> : <Login />} />
      <Route path="/register" element={<CreateAccount />} />
      <Route path="/check-email" element={<CheckEmail />} />
      <Route path="/confirm-email" element={<ConfirmEmail />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/reset-password" element={<ResetPassword />} />
      <Route path="/set-password" element={<ResetPassword />} />
      <Route path="/logout" element={<Logout />} />
      <Route path="/logged-out" element={<LoggedOut />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
