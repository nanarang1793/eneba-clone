import { Outlet } from "react-router";
import { Header } from "./Header";

/** Shared page frame: header on top, the current route's page in <Outlet />. */
export function Layout() {
  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6">
        <Outlet />
      </main>
      <footer className="border-t border-line py-6 text-center text-xs text-muted">
        Eneba clone, built for the Eneba Intern Assignment 2027. Not affiliated with Eneba. Prices and keys are fake.
      </footer>
    </div>
  );
}
