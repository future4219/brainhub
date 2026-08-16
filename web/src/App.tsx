import { BrowserRouter, Route, Routes } from "react-router-dom";

import { BrainDetail } from "@/components/pages/BrainDetail";
import { BrainList } from "@/components/pages/BrainList";
import { CreateBrain } from "@/components/pages/CreateBrain";
import { Error404 } from "@/components/pages/Error404";
import { Invitation } from "@/components/pages/Invitation";
import { Login } from "@/components/pages/Login";
import { Register } from "@/components/pages/Register";
import { appUrl } from "@/config/url";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path={appUrl.brainList} element={<BrainList />} />
        <Route path={appUrl.login} element={<Login />} />
        <Route path={appUrl.register} element={<Register />} />
        <Route path={appUrl.createBrain} element={<CreateBrain />} />
        <Route path={appUrl.invitation} element={<Invitation />} />
        <Route path={appUrl.brainDetail} element={<BrainDetail />} />
        <Route path="*" element={<Error404 />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
