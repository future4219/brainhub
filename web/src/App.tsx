import { BrowserRouter, Route, Routes } from "react-router-dom";

import { BrainDetail } from "@/components/pages/BrainDetail";
import { BrainList } from "@/components/pages/BrainList";
import { BrainPage } from "@/components/pages/BrainPage";
import { CreateBrain } from "@/components/pages/CreateBrain";
import { Connections } from "@/components/pages/Connections";
import { Error404 } from "@/components/pages/Error404";
import { Invitation } from "@/components/pages/Invitation";
import { Login } from "@/components/pages/Login";
import { OAuthConsent } from "@/components/pages/OAuthConsent";
import { PageEditor } from "@/components/pages/PageEditor";
import { Register } from "@/components/pages/Register";
import { appUrl } from "@/config/url";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path={appUrl.brainList} element={<BrainList />} />
        <Route path={appUrl.connections} element={<Connections />} />
        <Route path={appUrl.login} element={<Login />} />
        <Route path={appUrl.register} element={<Register />} />
        <Route path={appUrl.createBrain} element={<CreateBrain />} />
        <Route path={appUrl.invitation} element={<Invitation />} />
        <Route path={appUrl.oauthAuthorize} element={<OAuthConsent />} />
        <Route path={appUrl.newPage} element={<PageEditor />} />
        <Route path={appUrl.editPage} element={<PageEditor />} />
        <Route path={appUrl.brainPage} element={<BrainPage />} />
        <Route path={appUrl.brainDetail} element={<BrainDetail />} />
        <Route path="*" element={<Error404 />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
