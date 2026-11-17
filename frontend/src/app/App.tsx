import { Link, Navigate, Route, Routes } from "react-router";
import { Layout } from "../components/layout/Layout";
import { StorePage } from "../features/store/StorePage";

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        {/* The data-driven homepage comes on day 6; until then / opens the store. */}
        <Route index element={<Navigate to="/store" replace />} />
        <Route path="store" element={<StorePage />} />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  );
}

function NotFound() {
  return (
    <div className="py-24 text-center">
      <h1 className="text-3xl font-bold">Page not found</h1>
      <Link to="/store" className="mt-4 inline-block text-accent hover:underline">
        Go to the store
      </Link>
    </div>
  );
}
