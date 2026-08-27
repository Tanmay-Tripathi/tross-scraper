import { Link, Route, Routes } from "react-router-dom";
import RootLayout from "@/app/layout";
import HomePage from "@/app/page";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ROUTES } from "@/types/routes";
import ErrorHandler from "@/utils/ErrorHandler";

function NotFoundPage() {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-4">
      <h1 className="font-bold text-4xl">404</h1>
      <p className="text-muted-foreground">
        The page you are looking for does not exist.
      </p>
      <Link className="text-primary underline" to={ROUTES.HOME}>
        Back home
      </Link>
    </div>
  );
}

export default function App() {
  return (
    <TooltipProvider delayDuration={300}>
      <ErrorHandler componentName="App">
        <RootLayout>
          <Routes>
            <Route path={ROUTES.HOME} element={<HomePage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </RootLayout>
      </ErrorHandler>
    </TooltipProvider>
  );
}
